package state

import (
	"context"
	"fmt"

	"github.com/coinbase/step/handler"
	"github.com/coinbase/step/jsonpath"
	"github.com/coinbase/step/utils/to"
)

type ActionState struct {
	stateStr // Include Defaults

	Type       *string
	Comment    *string `json:",omitempty"`
	ActionName *string `json:",omitempty"`

	InputPath  *jsonpath.Path `json:",omitempty"`
	OutputPath *jsonpath.Path `json:",omitempty"`
	ResultPath *jsonpath.Path `json:",omitempty"`
	Parameters interface{}    `json:",omitempty"`

	Catch []*Catcher `json:",omitempty"`
	Retry []*Retrier `json:",omitempty"`

	ActionHandler handler.ActionHandler `json:"-"`

	Next *string `json:",omitempty"`
	End  *bool   `json:",omitempty"`

	TimeoutSeconds   int `json:",omitempty"`
	HeartbeatSeconds int `json:",omitempty"`
}

func (s *ActionState) SetActionHandler(resourcefn interface{}) {
	s.ActionHandler = resourcefn.(handler.ActionHandler)
}

func (s *ActionState) process(ctx context.Context, input interface{}) (interface{}, *string, error) {
	params := handler.Params(input.(map[string]interface{}))
	result, err := s.ActionHandler(ctx, *(s.ActionName), params)

	if err != nil {
		return nil, nil, err
	}

	result, err = to.FromJSON(result)

	if err != nil {
		return nil, nil, err
	}

	return result, nextState(s.Next, s.End), nil
}

// Input must include the Action name in $.Action
func (s *ActionState) Execute(ctx context.Context, input interface{}) (output interface{}, next *string, err error) {
	return processError(s,
		processCatcher(s.Catch,
			processRetrier(s.Name(), s.Retry,
				inputOutput(
					s.InputPath,
					s.OutputPath,
					withParams(
						s.Parameters,
						result(s.ResultPath, s.process),
					),
				),
			),
		),
	)(ctx, input)
}

func (s *ActionState) Validate() error {
	s.SetType(to.Strp("Action"))

	if err := ValidateNameAndType(s); err != nil {
		return fmt.Errorf("%v %v", errorPrefix(s), err)
	}

	if err := endValid(s.Next, s.End); err != nil {
		return fmt.Errorf("%v %v", errorPrefix(s), err)
	}

	if s.ActionHandler != nil {
		if err := handler.ValidateActionHandler(s.ActionHandler); err != nil {
			return err
		}
	}

	if err := catchValid(s.Catch); err != nil {
		return err
	}

	if err := retryValid(s.Retry); err != nil {
		return err
	}

	return nil
}

func (s *ActionState) SetType(t *string) {
	s.Type = t
}

func (s *ActionState) GetType() *string {
	return s.Type
}
