package consumer

import (
	"context"
	. "fmt"
	_ "net/http/pprof"

	prod "example.com/usagefacts/producer"
)

func Send(ctx context.Context, writer prod.Writer) error {
	Println("sending")
	resp := &prod.Response{Value: "ok"}
	return writer.WriteThing(ctx, resp)
}

func SendConcrete(ctx context.Context) error {
	service := prod.Service{}
	return service.WriteThing(ctx, prod.NewResponse("ok"))
}
