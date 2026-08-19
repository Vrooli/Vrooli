package documents

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	documentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/documents"
	"persona/internal/handoffs"
	"persona/internal/journal"
	"persona/internal/personas"
)

type documentPersonaFunc func(context.Context, string) (personas.Persona, error)

func (f documentPersonaFunc) Get(ctx context.Context, id string) (personas.Persona, error) {
	return f(ctx, id)
}

type documentHandoffFunc func(context.Context, string) (handoffs.Handoff, error)

func (f documentHandoffFunc) Get(ctx context.Context, id string) (handoffs.Handoff, error) {
	return f(ctx, id)
}

type documentJournal struct{ entries []journal.Entry }

func (j *documentJournal) Append(_ context.Context, entry journal.Entry) (journal.Entry, error) {
	j.entries = append(j.entries, entry)
	return entry, nil
}

func (j *documentJournal) List(context.Context, string, int) ([]journal.Entry, error) {
	return j.entries, nil
}

func documentPersona() personas.Persona {
	return personas.Persona{ID: "persona-1", Kind: personas.KindPersonal, Status: personas.StatusActive}
}

func documentHandoff() handoffs.Handoff {
	return handoffs.Handoff{ID: "handoff-1", PersonaID: "persona-1", State: handoffs.StateAwaitingHuman, Checkpoint: handoffs.Checkpoint{RequiredDocumentIDs: []string{"document-1"}}}
}

func TestBindingFailsClosedButExistingBindingsRemainListable(t *testing.T) {
	// [REQ:PSN-P0-008] this domain stores references and refuses when document-manager is unavailable.
	repo := &FakeRepository{
		ListFunc: func(context.Context, string) ([]Binding, error) {
			return []Binding{{ID: "binding-1", PersonaID: "persona-1", DocumentID: "document-1"}}, nil
		},
		CreateFunc: func(context.Context, Binding) (Binding, error) { return Binding{}, nil },
	}
	j := &documentJournal{}
	service := NewService(repo, documentPersonaFunc(func(context.Context, string) (personas.Persona, error) { return documentPersona(), nil }), documentHandoffFunc(func(context.Context, string) (handoffs.Handoff, error) { return documentHandoff(), nil }), FakeAuthority{
		CheckFunc:   func(context.Context) error { return errors.New("document-manager down") },
		ReleaseFunc: func(context.Context, string, string) (string, error) { return "", nil },
	}, j, nil)
	_, err := service.Bind(context.Background(), BindingInput{PersonaID: "persona-1", DocumentID: "document-1", DocumentKind: "passport"})
	if !errors.Is(err, ErrDocumentAuthorityUnavailable) {
		t.Fatalf("Bind() error = %v", err)
	}
	bindings, err := service.List(context.Background(), "persona-1")
	if err != nil || len(bindings) != 1 {
		t.Fatalf("List() = %#v, %v", bindings, err)
	}
	if len(j.entries) != 1 || j.entries[0].Outcome != "refused" {
		t.Fatalf("binding refusal journal = %#v", j.entries)
	}
}

func TestReleaseRequiresNamedMatchingHandoff(t *testing.T) {
	// [REQ:PSN-P0-009] release is server-bound to a pre-declared handoff.
	repo := &FakeRepository{
		GetFunc: func(_ context.Context, personaID, documentID string) (Binding, error) {
			return Binding{ID: "binding-1", PersonaID: personaID, DocumentID: documentID}, nil
		},
	}
	j := &documentJournal{}
	service := NewService(repo, documentPersonaFunc(func(context.Context, string) (personas.Persona, error) { return documentPersona(), nil }), documentHandoffFunc(func(_ context.Context, id string) (handoffs.Handoff, error) {
		if id != "handoff-1" {
			return handoffs.Handoff{ID: id, PersonaID: "persona-1", Checkpoint: handoffs.Checkpoint{RequiredDocumentIDs: []string{"other-document"}}}, nil
		}
		return documentHandoff(), nil
	}), FakeAuthority{
		CheckFunc:   func(context.Context) error { return nil },
		ReleaseFunc: func(context.Context, string, string) (string, error) { return "release-1", nil },
	}, j, nil)
	_, err := service.ReleaseIntoHandoff(context.Background(), ReleaseInput{PersonaID: "persona-1", DocumentID: "document-1"})
	if !errors.Is(err, ErrMissingHandoff) {
		t.Fatalf("unnamed handoff error = %v", err)
	}
	_, err = service.ReleaseIntoHandoff(context.Background(), ReleaseInput{PersonaID: "persona-1", DocumentID: "document-1", HandoffID: "handoff-other"})
	if !errors.Is(err, ErrHandoffMismatch) {
		t.Fatalf("mismatched handoff error = %v", err)
	}
	release, err := service.ReleaseIntoHandoff(context.Background(), ReleaseInput{PersonaID: "persona-1", DocumentID: "document-1", HandoffID: "handoff-1"})
	if err != nil || release.ID != "release-1" {
		t.Fatalf("ReleaseIntoHandoff() = %#v, %v", release, err)
	}
}

func TestDocumentDescriptorHasNoContentReadPath(t *testing.T) {
	// [REQ:PSN-P0-008] the generated contract structurally exposes references and release only.
	methods := documentsv1.File_persona_v1_documents_documents_proto.Services().ByName("DocumentsService").Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		name := strings.ToLower(string(method.Name()))
		if strings.Contains(name, "content") || strings.Contains(name, "bytes") || strings.Contains(name, "download") {
			t.Fatalf("document content method exposed: %s", method.Name())
		}
		fields := method.Output().Fields()
		for j := 0; j < fields.Len(); j++ {
			fieldName := strings.ToLower(string(fields.Get(j).Name()))
			if strings.Contains(fieldName, "content") || strings.Contains(fieldName, "bytes") || strings.Contains(fieldName, "blob") {
				t.Fatalf("document content field exposed: %s", fields.Get(j).Name())
			}
		}
	}
}

func TestDocumentHealthSurfacesExpiredBindingAndAuthorityFailure(t *testing.T) {
	// [REQ:PSN-P1-005] expired custody references and an unreachable document authority are both blocking findings.
	service := NewService(&FakeRepository{ListFunc: func(context.Context, string) ([]Binding, error) {
		return []Binding{{DocumentID: "document-1", ValidUntil: time.Now().Add(-time.Minute)}}, nil
	}}, documentPersonaFunc(func(context.Context, string) (personas.Persona, error) { return documentPersona(), nil }), documentHandoffFunc(func(context.Context, string) (handoffs.Handoff, error) { return documentHandoff(), nil }), FakeAuthority{
		CheckFunc:   func(context.Context) error { return errors.New("document-manager down") },
		ReleaseFunc: func(context.Context, string, string) (string, error) { return "", nil },
	}, nil, nil)
	findings, err := service.CheckHealth(context.Background(), "persona-1")
	if err != nil || len(findings) != 2 {
		t.Fatalf("document health = %#v, %v", findings, err)
	}
}
