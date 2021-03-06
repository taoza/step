// Lambda Handler Data Structures and types
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime/debug"

	"github.com/coinbase/step/errors"
)

///////////
// TYPES
///////////

// TaskHandlers maps a Task Name String to a function <pre>ahsufasiu</pre>
type TaskHandlers map[string]interface{}

// TaskReflection caches lots of the reflected values from the Task functions in order to speed up calls
type TaskReflection struct {
	Handler   reflect.Value
	Type      reflect.Type
	EventType reflect.Type
}

// CreateTaskReflection creates a TaskReflection from a handler function
func CreateTaskReflection(handlerSymbol interface{}) TaskReflection {
	handlerType := reflect.TypeOf(handlerSymbol)

	return TaskReflection{
		Handler:   reflect.ValueOf(handlerSymbol),
		EventType: handlerType.In(1),
	}
}

// Tasks returns all Task names from a TaskHandlers Map
func (t *TaskHandlers) Tasks() []string {
	keys := []string{}
	for key, _ := range *t {
		keys = append(keys, key)
	}
	return keys
}

// TaskHandlers Returns a map of TaskReflections from TaskHandlers
func (t *TaskHandlers) Reflect() map[string]TaskReflection {
	ref := map[string]TaskReflection{}
	for name, handler := range *t {
		ref[name] = CreateTaskReflection(handler)
	}
	return ref
}

// TaskHandlers validates all handlers in a TaskHandlers map
func (t *TaskHandlers) Validate() error {
	// Each
	for name, handler := range *t {
		if err := ValidateHandler(handler); err != nil {
			return &TaskError{err.Error(), &name, t.Tasks()}
		}
	}
	return nil
}

// ValidateHandler checks a handler is a function with the correct arguments and return values
func ValidateHandler(handlerSymbol interface{}) error {
	if handlerSymbol == nil {
		return fmt.Errorf("Handler nil")
	}

	handlerType := reflect.TypeOf(handlerSymbol)

	if handlerType.Kind() != reflect.Func {
		return fmt.Errorf("handler kind %s is not %s", handlerType.Kind(), reflect.Func)
	}

	err := validateArguments(handlerType)
	if err != nil {
		return err
	}

	return nil
}

func validateArguments(handler reflect.Type) error {
	if handler.NumIn() != 2 {
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

//////
// RawMessage
//////

// RawMessage is the struct passed to the Lambda Handler
// It contains the name of the Task and the Inputs Raw message
type RawMessage struct {
	Task       *string
	Input      json.RawMessage
	Parameters json.RawMessage
}

///////////
// Errors
///////////

// TaskError is a error type a task function may throw handling it in the state machine is a good idea
type TaskError struct {
	ErrorString string
	Task        *string
	Tasks       []string
}

func (t *TaskError) Error() string {
	for_task := ""
	with_taskmap := ""

	if t.Task != nil {
		for_task = fmt.Sprintf("(%v)", *t.Task)
	}

	if t.Tasks != nil {
		with_taskmap = fmt.Sprintf(" : %v", t.Tasks)
	}

	return fmt.Sprintf("TaskError%v%v: %v", for_task, with_taskmap, t.ErrorString)
}

///////////
// FUNCTIONS
///////////

// CreateHandler returns the handler passed to the lambda.Start function
func CreateHandler(tm *TaskHandlers) (func(context context.Context, input *RawMessage) (interface{}, error), error) {
	if err := tm.Validate(); err != nil {
		return nil, err
	}

	// This does most reflection before the run handler,
	// that way there is less reflection in the main call
	reflections := tm.Reflect()

	handler := func(ctx context.Context, input *RawMessage) (interface{}, error) {
		// Find Resource Handler
		task_name := input.Task
		if task_name == nil {
			return nil, &TaskError{"Nil Task In Message", nil, nil}
		}

		reflection, ok := reflections[*task_name]

		if !ok {
			return nil, &TaskError{"Cannot Find Task", task_name, tm.Tasks()}
		}

		return CallHandler(reflection, ctx, input.Input, input.Parameters)
	}

	return handler, nil
}

func recoveryError(r interface{}) error {
	switch x := r.(type) {
	case string:
		return errors.PanicError{x}
	case error:
		return errors.PanicError{x.Error()}
	default:
		return errors.PanicError{fmt.Sprintf("Unknown %v", x)}
	}
}

// HANDLERS

// CallHandler calls a TaskReflections Handler with the correct objects using reflection
// Mostly borrowed from the aws-lambda-go package
func CallHandler(reflection TaskReflection, ctx context.Context, input []byte, rawParams []byte) (ret interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovering", r, fmt.Sprintf("%s\n", debug.Stack()))
			err = recoveryError(r)
			ret = nil
		}
	}()

	event := reflect.New(reflection.EventType)

	if err = json.Unmarshal(input, event.Interface()); err != nil {
		return nil, errors.UnmarshalError{err.Error()}
	}

	// Get Type of Function Input
	var args []reflect.Value
	if ctx == nil {
		ctx = context.Background()
	}

	params := map[string]interface{}{}
	json.Unmarshal(rawParams, &params)
	ctx = context.WithValue(ctx, "Params", params)

	args = append(args, reflect.ValueOf(ctx))
	args = append(args, event.Elem())

	response := reflection.Handler.Call(args)

	if errVal, ok := response[1].Interface().(error); ok {
		err = errVal
	}
	ret = response[0].Interface()

	return ret, err
}

// CallHandlerFunction does reflection inline and should only be used for testing
func CallHandlerFunction(handlerSymbol interface{}, ctx context.Context, input interface{}) (interface{}, error) {

	if err := ValidateHandler(handlerSymbol); err != nil {
		return nil, err
	}

	raw_json, err := json.Marshal(input)

	if err != nil {
		return nil, fmt.Errorf("JSON Marshall Error: %v", err)
	}

	reflection := CreateTaskReflection(handlerSymbol)
	return CallHandler(reflection, ctx, raw_json, nil)
}
