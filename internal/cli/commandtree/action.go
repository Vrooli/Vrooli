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

func WriteHelp(w io.Writer, text string) {
	_, _ = io.WriteString(w, text)
	if text == "" || text[len(text)-1] != '\n' {
		_, _ = io.WriteString(w, "\n")
	}
}

func HandleHelp(w io.Writer, err error) bool {
	helpErr, ok := err.(HelpError)
	if !ok {
		return false
	}
	WriteHelp(w, helpErr.HelpText())
	return true
}

func ExecuteAction[Req any, Resp any](stdout io.Writer, args []string, action Action[Req, Resp]) error {
	req, err := action.Parse(args)
	if err != nil {
		if HandleHelp(stdout, err) {
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
