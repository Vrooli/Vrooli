package prompt_manager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/api-core/discovery"
)

func TestSkillContentFetchesBody(t *testing.T) {
	const id = "progress"
	const body = "# Progress\n\nAdvance the operational progress log.\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/skills/"+id) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"progress","content":` + jsonString(body) + `}`))
			return
		}
		http.Error(w, "unexpected: "+r.URL.Path, http.StatusNotFound)
	}))
	defer srv.Close()

	a := NewSkillContentRESTAdapter(Options{Resolver: discovery.NewStaticResolver(srv.URL)})
	got, err := a.SkillContent(context.Background(), id)
	if err != nil {
		t.Fatalf("SkillContent: %v", err)
	}
	if got != body {
		t.Fatalf("content = %q, want %q", got, body)
	}
}

func TestSkillContentEmptyIDErrors(t *testing.T) {
	a := NewSkillContentRESTAdapter(Options{Resolver: discovery.NewStaticResolver("http://unused")})
	if _, err := a.SkillContent(context.Background(), "   "); err == nil {
		t.Fatalf("empty id should error")
	}
}

func TestSkillContentNotFoundErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Skill not found", http.StatusNotFound)
	}))
	defer srv.Close()

	a := NewSkillContentRESTAdapter(Options{Resolver: discovery.NewStaticResolver(srv.URL)})
	if _, err := a.SkillContent(context.Background(), "missing"); err == nil {
		t.Fatalf("404 should surface as error")
	}
}

// jsonString returns a minimally-escaped JSON string literal for the test
// body (which contains newlines and a leading '#').
func jsonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
