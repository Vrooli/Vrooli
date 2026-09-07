package smoketest

import (
	"context"
	"reflect"
	"testing"
)

func TestValidationRendererEnvPropagatesRendererAndAPIURLs(t *testing.T) {
	service := &DefaultService{
		rendererURLResolver: func(context.Context, string) (string, error) { return "http://127.0.0.1:23230", nil },
		apiURLResolver:      func(context.Context, string) (string, error) { return "http://localhost:23154", nil },
	}
	got := service.validationRendererEnv(context.Background(), "hello-desktop")
	want := []string{
		"VROOLI_VALIDATION_RENDERER_URL=http://127.0.0.1:23230",
		"VROOLI_VALIDATION_API_URL=http://localhost:23154",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("validation environment = %#v, want %#v", got, want)
	}
}

func TestValidationRendererEnvKeepsAvailableURLWhenOtherResolverFails(t *testing.T) {
	service := &DefaultService{
		rendererURLResolver: func(context.Context, string) (string, error) { return "http://127.0.0.1:23230", nil },
		apiURLResolver:      func(context.Context, string) (string, error) { return "", context.Canceled },
	}
	got := service.validationRendererEnv(context.Background(), "hello-desktop")
	if !reflect.DeepEqual(got, []string{"VROOLI_VALIDATION_RENDERER_URL=http://127.0.0.1:23230"}) {
		t.Fatalf("validation environment = %#v", got)
	}
}
