package controller

import (
	"github.com/jmainguy/coastie-operator/pkg/controller/coastieservice"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, coastieservice.Add)
}
