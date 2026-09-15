package cliapp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestProtoBindingsRendererReceivesConcreteMessageThroughDispatcher(t *testing.T) {
	service, method := registerRendererTestService(t)
	path := "/test.cli_bindings.v1.EchoService/Echo"
	handler := connect.NewUnaryHandlerSimple(path, func(_ context.Context, request *dynamicpb.Message) (*dynamicpb.Message, error) {
		response := dynamicpb.NewMessage(method.Output())
		field := response.Descriptor().Fields().ByName("value")
		response.Set(field, request.Get(request.Descriptor().Fields().ByName("value")))
		return response, nil
	}, connect.WithSchema(method), connect.WithRequestInitializer(func(_ connect.Spec, message any) error {
		dynamic, ok := message.(*dynamicpb.Message)
		if !ok {
			return fmt.Errorf("request initializer received %T", message)
		}
		*dynamic = *dynamicpb.NewMessage(method.Input())
		return nil
	}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()

	app := &ScenarioApp{
		HTTPClient: cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{Override: server.URL}}),
		options:    ScenarioOptions{APIPrefix: "/"},
	}
	app.baseOptions = func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{Override: server.URL} }
	app.tokenSource = func() string { return "" }

	var received proto.Message
	bindings, err := ProtoBindings(app, service.FullName(), ProtoBindingOptions{
		Render: map[string]Renderer{
			"EchoService.Echo": func(_ RunContext, message proto.Message) error {
				received = message
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("ProtoBindings: %v", err)
	}
	ctx := NewTestRunContext(TestRunContextOptions{
		Core:   app,
		Schema: ArgSchema{Flags: []Flag{{Name: "value"}}},
		Flags:  map[string]string{"value": "through-dispatcher"},
	})
	if err := bindings["EchoService.Echo"](ctx); err != nil {
		t.Fatalf("binding: %v", err)
	}
	concrete, ok := received.(*wrapperspb.StringValue)
	if !ok {
		t.Fatalf("renderer received %T, want *wrapperspb.StringValue", received)
	}
	if concrete.GetValue() != "through-dispatcher" {
		t.Fatalf("renderer received value %q", concrete.GetValue())
	}
}

func registerRendererTestService(t *testing.T) (protoreflect.ServiceDescriptor, protoreflect.MethodDescriptor) {
	t.Helper()
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:       proto.String("test/cli_bindings.proto"),
		Package:    proto.String("test.cli_bindings.v1"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"google/protobuf/wrappers.proto"},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("EchoService"),
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name:       proto.String("Echo"),
				InputType:  proto.String(".google.protobuf.StringValue"),
				OutputType: proto.String(".google.protobuf.StringValue"),
			}},
		}},
	}, protoregistry.GlobalFiles)
	if err != nil {
		t.Fatalf("build test descriptor: %v", err)
	}
	if err := protoregistry.GlobalFiles.RegisterFile(file); err != nil {
		t.Fatalf("register test descriptor: %v", err)
	}
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName("test.cli_bindings.v1.EchoService")
	if err != nil {
		t.Fatalf("find test service: %v", err)
	}
	service := desc.(protoreflect.ServiceDescriptor)
	return service, service.Methods().ByName("Echo")
}
