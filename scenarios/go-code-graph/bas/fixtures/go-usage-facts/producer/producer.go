package producer

import "context"

type Response struct {
	Value string
}

type Writer interface {
	WriteThing(context.Context, *Response) error
}

type Service struct{}

func (Service) WriteThing(_ context.Context, _ *Response) error {
	return nil
}

func NewResponse(value string) *Response {
	return &Response{Value: value}
}
