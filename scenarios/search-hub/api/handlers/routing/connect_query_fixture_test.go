package routing_test

import (
	"context"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	handler "search-hub/handlers/routing"
)

func TestConnectQueryProtoJSONRoundTripsSharedTaskFixture(t *testing.T) {
	fixture, err := os.ReadFile("testdata/query-task-response.json")
	if err != nil {
		t.Fatal(err)
	}
	var response routingv1.QueryResponse
	if err := protojson.Unmarshal(fixture, &response); err != nil {
		t.Fatalf("decode shared fixture: %v", err)
	}
	logger, _ := connectxtest.NewLogger(t)
	path, handlerHTTP := routingconnect.NewRoutingServiceHandler(handler.NewConnectHandler(handler.Deps{
		Router: &fakeRouter{out: &response}, Logger: logger,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handlerHTTP})
	client := routingconnect.NewRoutingServiceClient(server.Client(), server.URL, connect.WithProtoJSON())
	got, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{Query: "fixture task"}))
	if err != nil {
		t.Fatalf("Query through the actual Connect handler: %v", err)
	}
	if !proto.Equal(&response, got.Msg) {
		t.Fatalf("Connect ProtoJSON response differs from fixture response\nwant: %v\n got: %v", &response, got.Msg)
	}
}
