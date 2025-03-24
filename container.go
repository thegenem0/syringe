package main

import (
	"reflect"
	"sync"
)

type ServiceLifetime int

const (
	// Transient services are re-created on each request
	Transient ServiceLifetime = iota
	// Scoped services are re-created on every scope
	Scoped
	// Singleton services live for the lifetime of the application
	Singleton
)

type DependencyRegistration struct {
	ServiceType    reflect.Type
	Implementation interface{}
	Factory        interface{}
	Instance       interface{}
	Lifetime       ServiceLifetime
}

type Container struct {
	registrations []DependencyRegistration
	mu            sync.RWMutex
}

func NewContainer() *Container {
	return &Container{
		registrations: make([]DependencyRegistration, 0),
	}
}

func (s *Container) AddSingleton(serviceType interface{}, implementation interface{}) *Container {
	return s.addService(serviceType, implementation, nil, nil, Singleton)
}

func (s *Container) AddSingletonByFactory(serviceType interface{}, factory interface{}) *Container {
	return s.addService(serviceType, nil, factory, nil, Singleton)
}

func (s *Container) AddSingletonInstance(serviceType interface{}, instance interface{}) *Container {
	return s.addService(serviceType, nil, nil, instance, Singleton)
}

func (s *Container) AddScoped(serviceType interface{}, implementation interface{}) *Container {
	return s.addService(serviceType, implementation, nil, nil, Scoped)
}

func (s *Container) AddScopedByFactory(serviceType interface{}, factory interface{}) *Container {
	return s.addService(serviceType, nil, factory, nil, Scoped)
}

func (s *Container) AddTransient(serviceType interface{}, implementation interface{}) *Container {
	return s.addService(serviceType, implementation, nil, nil, Transient)
}

func (s *Container) AddTransientByFactory(serviceType interface{}, factory interface{}) *Container {
	return s.addService(serviceType, nil, factory, nil, Transient)
}

func (s *Container) addService(
	serviceType interface{},
	implementation interface{},
	factory interface{},
	instance interface{},
	lifetime ServiceLifetime) *Container {

	s.mu.Lock()
	defer s.mu.Unlock()

	var serviceTypeObj reflect.Type
	if t, ok := serviceType.(reflect.Type); ok {
		serviceTypeObj = t
	} else {
		serviceTypeObj = reflect.TypeOf(serviceType).Elem()
	}

	s.registrations = append(s.registrations, DependencyRegistration{
		ServiceType:    serviceTypeObj,
		Implementation: implementation,
		Factory:        factory,
		Instance:       instance,
		Lifetime:       lifetime,
	})

	return s
}
