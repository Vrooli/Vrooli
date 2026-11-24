package facts

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	tsgraphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
	"google.golang.org/protobuf/proto"
)

type fixedURLResolver string

func (r fixedURLResolver) ResolveScenarioURLDefault(_ context.Context, scenario string) (string, error) {
	if scenario != tsProviderScenario {
		return "", io.ErrUnexpectedEOF
	}
	return string(r), nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestTypeScriptProviderThreadsImportProfileAndPreservesResponse(t *testing.T) {
	want := &tsgraphv1.ExtractResponse{ExtractionProfile: "imports", GraphHash: "import-graph"}
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		var got tsgraphv1.ExtractRequest
		if err := proto.Unmarshal(body, &got); err != nil {
			return nil, err
		}
		if got.GetProjectPath() != "/repo/ui/package.json" || got.GetProfile() != "imports" {
			return nil, errors.New("provider request did not preserve target and imports profile")
		}
		encoded, err := proto.Marshal(want)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/proto"}},
			Body:       io.NopCloser(bytes.NewReader(encoded)),
			Request:    request,
		}, nil
	})
	provider := &tsGraphProvider{
		resolver:   fixedURLResolver("http://typescript-code-graph.test"),
		httpClient: &http.Client{Transport: transport},
		language:   "typescript",
	}

	got, err := provider.ExtractForFamilies(context.Background(), &factsv1.ParseUnit{
		RootPath: "/repo/ui", ConfigPath: "/repo/ui/package.json",
	}, []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS})
	if err != nil {
		t.Fatal(err)
	}
	if got.GetExtractionProfile() != want.GetExtractionProfile() || got.GetGraphHash() != want.GetGraphHash() {
		t.Fatalf("provider output = %+v, want profile/hash %q/%q", got, want.GetExtractionProfile(), want.GetGraphHash())
	}
	if len(got.GetOmittedInformation()) != 3 {
		t.Fatalf("import profile omissions = %+v, want declarations, usage facts, and diagnostics", got.GetOmittedInformation())
	}
}
