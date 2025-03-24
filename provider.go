package main

import (
	"fmt"
	"reflect"
	"sync"
)

type InjectionProvider struct {
	root            *InjectionProvider
	registrations   []DependencyRegistration
	singletonCache  map[reflect.Type]interface{}
	scopedCache     map[reflect.Type]interface{}
	resolutionStack []reflect.Type
	mu              sync.RWMutex
}

func (p *InjectionProvider) DebugLogServices() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, reg := range p.registrations {
		fmt.Printf("%s: %s\n", reg.ServiceType.String(), reg.Implementation)
	}
}

func (s *Container) BuildServiceProvider() *InjectionProvider {
	s.mu.RLock()
	registrations := make([]DependencyRegistration, len(s.registrations))
	copy(registrations, s.registrations)
	s.mu.RUnlock()

	provider := &InjectionProvider{
		registrations:  registrations,
		singletonCache: make(map[reflect.Type]interface{}),
		scopedCache:    make(map[reflect.Type]interface{}),
	}
	provider.root = provider
	return provider
}

// CreateScope creates a new service scope
func (p *InjectionProvider) CreateScope() *InjectionProvider {
	return &InjectionProvider{
		root:           p.root,
		registrations:  p.registrations,
		singletonCache: p.root.singletonCache,              // Share singleton cache
		scopedCache:    make(map[reflect.Type]interface{}), // New scoped cache
	}
}

func (p *InjectionProvider) GetService(serviceType interface{}) (interface{}, error) {
	var typeObj reflect.Type

	if t, ok := serviceType.(reflect.Type); ok {
		typeObj = t
	} else {
		typeObj = reflect.TypeOf(serviceType).Elem()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.resolutionStack = make([]reflect.Type, 0)

	return p.resolveService(typeObj)
}

// RequireService resolves a service instance or panics if not found
func (p *InjectionProvider) RequireService(serviceType interface{}) interface{} {
	service, err := p.GetService(serviceType)
	if err != nil {
		panic(err)
	}
	return service
}

func (p *InjectionProvider) resolveService(serviceType reflect.Type) (interface{}, error) {
	// Check for circular dependencies
	for _, t := range p.resolutionStack {
		if t == serviceType {
			path := ""
			for _, p := range p.resolutionStack {
				path += fmt.Sprintf("%s -> ", p.String())
			}
			path += serviceType.String()
			return nil, fmt.Errorf("%w: %s", ErrCircularDependency, path)
		}
	}

	// Add to resolution stack
	p.resolutionStack = append(p.resolutionStack, serviceType)
	defer func() {
		// Remove from stack when done
		p.resolutionStack = p.resolutionStack[:len(p.resolutionStack)-1]
	}()

	// Check caches first
	if instance, ok := p.root.singletonCache[serviceType]; ok {
		return instance, nil
	}

	if instance, ok := p.scopedCache[serviceType]; ok {
		return instance, nil
	}

	// Find the registration
	var registration *DependencyRegistration
	for i := len(p.registrations) - 1; i >= 0; i-- {
		reg := &p.registrations[i]
		if reg.ServiceType == serviceType {
			registration = reg
			break
		}
	}

	if registration == nil {
		return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, serviceType.String())
	}

	var instance interface{}
	var err error

	switch {
	case registration.Instance != nil:
		// Return the pre-created instance
		instance = registration.Instance

	case registration.Factory != nil:
		// Call the factory function
		instance, err = p.callFactory(registration.Factory, serviceType)
		if err != nil {
			return nil, err
		}

	case registration.Implementation != nil:
		// Create a new instance of the implementation
		instance, err = p.createInstance(registration.Implementation)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("%w: no implementation for %s", ErrInvalidServiceType, serviceType.String())
	}

	// Cache the instance based on lifetime
	switch registration.Lifetime {
	case Singleton:
		p.root.singletonCache[serviceType] = instance
	case Scoped:
		p.scopedCache[serviceType] = instance
	}

	return instance, nil
}

func (p *InjectionProvider) callFactory(factory interface{}, serviceType reflect.Type) (interface{}, error) {
	factoryVal := reflect.ValueOf(factory)

	// Simple case: factory is a func(p *ServiceProvider) (T, error)
	if factoryVal.Type().NumIn() == 1 &&
		factoryVal.Type().In(0) == reflect.TypeOf(p) &&
		factoryVal.Type().NumOut() == 2 &&
		factoryVal.Type().Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {

		results := factoryVal.Call([]reflect.Value{reflect.ValueOf(p)})

		if !results[1].IsNil() {
			return nil, results[1].Interface().(error)
		}

		return results[0].Interface(), nil
	}

	return nil, fmt.Errorf("invalid factory function for %s", serviceType.String())
}

// createInstance creates a new instance of a service
func (p *InjectionProvider) createInstance(implementation interface{}) (interface{}, error) {
	var implType reflect.Type

	if t, ok := implementation.(reflect.Type); ok {
		implType = t
	} else {
		implType = reflect.TypeOf(implementation)
	}

	// Make sure it's a pointer to struct
	if implType.Kind() == reflect.Ptr && implType.Elem().Kind() == reflect.Struct {
		// Create a new instance
		instance := reflect.New(implType.Elem()).Interface()

		// Inject dependencies (simple property injection)
		if err := p.injectDependencies(instance); err != nil {
			return nil, err
		}

		return instance, nil
	}

	return nil, fmt.Errorf("implementation must be a pointer to struct: %s", implType.String())
}

// injectDependencies performs property injection
func (p *InjectionProvider) injectDependencies(instance interface{}) error {
	val := reflect.ValueOf(instance).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !fieldVal.CanSet() {
			continue
		}

		// Only try to inject interface types
		if field.Type.Kind() == reflect.Interface {
			// See if we can resolve this service
			service, err := p.resolveService(field.Type)
			if err == nil {
				fieldVal.Set(reflect.ValueOf(service))
			}
			// Silently continue if not found as property injection is optional
			// might need to add some error handling around this
		}
	}

	return nil
}

func (p *InjectionProvider) Dispose() {
	panic("not implemented")
}
