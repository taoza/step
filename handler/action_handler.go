package handler

import (
	"context"
	"fmt"
	"reflect"
	"runtime/debug"
)

type (
	Params         map[string]interface{}
	ActionHandler  func(ctx context.Context, actionName string, params Params) (interface{}, error)
	ActionHandlers map[string]ActionHandler
)

// ValidateActionHandler checks a handler is a function with the correct arguments and return values
func ValidateActionHandler(handlerSymbol interface{}) error {
	if handlerSymbol == nil {
		return fmt.Errorf("Handler nil")
	}

	handler := reflect.TypeOf(handlerSymbol)

	if handler.Kind() != reflect.Func {
		return fmt.Errorf("handler kind %s is not %s", handler.Kind(), reflect.Func)
	}

	if handler.NumIn() != 3 {
		debug.PrintStack()
		return fmt.Errorf("handlers must take two arguments, but handler takes %d", handler.NumIn())
	}

	if handler.NumOut() != 2 {
		return fmt.Errorf("handlers must return two arguments, but handler returns %d", handler.NumOut())
	}

	first_in := handler.In(0)
	second_out := handler.Out(1)

	// First Argument implements Context
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if !first_in.Implements(contextType) {
		return fmt.Errorf("handlers first argument must implement context.Context")
	}

	// Second Argument must be error
	errorInterface := reflect.TypeOf((*error)(nil)).Elem()
	if !second_out.Implements(errorInterface) {
		return fmt.Errorf("handlers second return value must be error")
	}

	return nil
}
