package journal

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestAppendRequiresIdentityAndListIsTheOnlyRead(t *testing.T) {
	var appended Entry
	service := NewService(FakeRepository{
		AppendFunc: func(_ context.Context, entry Entry) (Entry, error) {
			appended = entry
			return entry, nil
		},
		ListFunc: func(_ context.Context, _ string, _ int) ([]Entry, error) { return []Entry{appended}, nil },
	})

	// [REQ:PSN-P0-011] every journal row names a persona and an action verb.
	if _, err := service.Append(context.Background(), Entry{Verb: "act_as_granted"}); !errors.Is(err, ErrMissingPersona) {
		t.Fatalf("missing persona error = %v", err)
	}
	if _, err := service.Append(context.Background(), Entry{PersonaID: "persona-1"}); !errors.Is(err, ErrMissingVerb) {
		t.Fatalf("missing verb error = %v", err)
	}
	entry, err := service.Append(context.Background(), Entry{PersonaID: "persona-1", Actor: "agent", Verb: "act_as_granted", RunID: "run-1", AuthorisingHuman: "human-1"})
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if entry.PersonaID != "persona-1" || appended.RunID != "run-1" || appended.AuthorisingHuman != "human-1" {
		t.Fatalf("journal entry lost attribution: %#v", appended)
	}

	// [REQ:PSN-P0-011] corrections are new entries; the service has no rewrite/delete verb.
	serviceType := reflect.TypeOf((*Service)(nil)).Elem()
	for i := 0; i < serviceType.NumMethod(); i++ {
		if serviceType.Method(i).Name != "Append" && serviceType.Method(i).Name != "List" {
			t.Fatalf("unexpected journal method: %s", serviceType.Method(i).Name)
		}
	}
	if strings.Contains(strings.ToUpper(Schema()), "UPDATE ") || strings.Contains(strings.ToUpper(Schema()), "DELETE ") {
		t.Fatal("journal schema contains a rewrite/delete statement")
	}
}
