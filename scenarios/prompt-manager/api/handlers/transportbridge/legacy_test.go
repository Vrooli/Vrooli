package transportbridge

import (
	"context"
	"math"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	agentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/agents"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestMaskedBodySelectsFieldMaskPaths(t *testing.T) {
	message := &agentsv1.AgentInput{
		DisplayName: "Ada",
		Description: "ignored",
		FileOrder:   []string{"SOUL.md", "TOOLS.md"},
	}

	body, err := MaskedBody(message, []string{"display_name", "file_order"})
	if err != nil {
		t.Fatal(err)
	}
	if got := body["displayName"]; got != "Ada" {
		t.Fatalf("displayName = %#v, want Ada", got)
	}
	if _, ok := body["description"]; ok {
		t.Fatal("description was not selected by the field mask")
	}
	if _, ok := body["fileOrder"]; !ok {
		t.Fatal("fileOrder was not selected after snake_case conversion")
	}
}

func TestProtoBodyReportsMarshalFailure(t *testing.T) {
	message := &structpb.Struct{Fields: map[string]*structpb.Value{
		"invalid": structpb.NewNumberValue(math.NaN()),
	}}
	if _, err := ProtoBody(message); err == nil {
		t.Fatal("ProtoBody accepted a non-JSON numeric value")
	}
}

func TestInvokePreservesHeadersVarsAndMapsErrors(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vrooli-Attribution") != "actor=test" {
			http.Error(w, "missing attribution", http.StatusBadRequest)
			return
		}
		if got := mux.Vars(r)["id"]; got != "missing" {
			http.Error(w, "missing route variable", http.StatusBadRequest)
			return
		}
		http.Error(w, "not here", http.StatusNotFound)
	}
	headers := make(http.Header)
	headers.Set("X-Vrooli-Attribution", "actor=test")
	_, err := Invoke(context.Background(), headers, handler, http.MethodGet, "/items/missing", nil, map[string]string{"id": "missing"})
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want %v (err=%v)", connect.CodeOf(err), connect.CodeNotFound, err)
	}
}

func TestMaskedBodyDefaultsRequireExplicitPresence(t *testing.T) {
	message := &teamsv1.TeamInput{}
	body, err := MaskedBody(message, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 {
		t.Fatalf("unmasked absent defaults must not become mutations: %#v", body)
	}
	body, err = MaskedBody(message, []string{"purpose", "effort_refs", "enabled"})
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := body["purpose"]; !ok || value != "" {
		t.Fatalf("explicit string clear missing: %#v", body)
	}
	if value, ok := body["enabled"]; !ok || value != false {
		t.Fatalf("explicit false missing: %#v", body)
	}
	if value, ok := body["effortRefs"]; !ok || len(value.([]any)) != 0 {
		t.Fatalf("explicit list clear missing: %#v", body)
	}
	if _, ok := body["lifetime"]; ok {
		t.Fatal("unselected default became a mutation")
	}
}
