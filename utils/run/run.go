// run takes arguments
package run

import (
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/coinbase/step/handler"
	"github.com/coinbase/step/machine"
	"github.com/coinbase/step/utils/is"
	"github.com/coinbase/step/utils/to"
)

// Exec returns a function that will execute the state machine
func Exec(state_machine *machine.StateMachine, err error) func(*string) (output string, err error) {
	if err != nil {
		return func(input *string) (output string, err error) {
			return "", err
		}
	}

	return func(input *string) (output string, err error) {
		if is.EmptyStr(input) {
			input = to.Strp("{}")
		}

		exec, err := state_machine.Execute(input)
		if err != nil {
			return "", err
		}

		return exec.OutputJSON, nil
	}
}

// JSON prints a state machine as JSON
func JSON(state_machine *machine.StateMachine, err error) {
	if err != nil {
		fmt.Println("ERROR", err)
	}

	json, err := to.PrettyJSON(state_machine)

	if err != nil {
		fmt.Println("ERROR", err)
	}

	fmt.Println(string(json))
	os.Exit(0)
}

// LambdaTasks takes task functions and and executes as a lambda
func LambdaTasks(task_functions *handler.TaskHandlers) {
	handler, err := handler.CreateHandler(task_functions)

	if err != nil {
		fmt.Println("ERROR", err)
	}

	lambda.Start(handler)

	fmt.Println("ERROR: lambda.Start returned, but should have blocked")
}
