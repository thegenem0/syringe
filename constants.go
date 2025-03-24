package syringe

import (
	"errors"
)

var (
	ErrServiceNotFound    = errors.New("service not found")
	ErrCircularDependency = errors.New("circular dependency detected")
	ErrInvalidServiceType = errors.New("invalid service type")
)
