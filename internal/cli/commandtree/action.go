package commandtree

import (
	"io"
)

type HelpError interface {
	error
	HelpText() string
}

type Action[Req any, Resp any] struct {
	Parse   func(args []string) (Req, error)
	Execute func(req Req) (Resp, error)
	Render  func(w io.Writer, resp Resp) error
}

func ExecuteAction[Req any, Resp any](stdout io.Writer, args []string, action Action[Req, Resp]) error {
	req, err := action.Parse(args)
	if err != nil {
		if helpErr, ok := err.(HelpError); ok {
			_, _ = io.WriteString(stdout, helpErr.HelpText())
			if text := helpErr.HelpText(); text == "" || text[len(text)-1] != '\n' {
				_, _ = io.WriteString(stdout, "\n")
			}
			return nil
		}
		return err
	}
	resp, err := action.Execute(req)
	if err != nil {
		return err
	}
	return action.Render(stdout, resp)
}
