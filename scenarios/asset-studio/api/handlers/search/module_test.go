package search

import (
	core "asset-studio/internal/studio"
	"connectrpc.com/connect"
	"context"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/search"
	"testing"
)

type memoryStore struct{ state *core.Studio }

func (m memoryStore) Load(context.Context) (*core.Studio, error) { return m.state, nil }
func (m memoryStore) Save(context.Context, *core.Studio) error   { return nil }
func TestSearchExposesOnlyIdentityAndReleasedMetadata(t *testing.T) {
	s := core.New()
	s.Identities["identity-1"] = core.Identity{ID: "identity-1", Name: "Slate Console", Kind: core.Product, Traits: map[string]string{"finish": "slate"}}
	s.Assets["released"] = &core.Asset{ID: "released", Status: core.Released, AltText: "Slate launch visual", Disclosure: "AI-generated", MediaType: "image/png", DerivationOperation: "image", BlobKey: "image-tools://secret-output"}
	s.Assets["draft"] = &core.Asset{ID: "draft", Status: core.Produced, AltText: "Slate draft", BlobKey: "image-tools://unreviewed"}
	h := handler{store: memoryStore{state: s}}
	resp, err := h.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "slate"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.Results) != 2 {
		t.Fatalf("results=%#v", resp.Msg.Results)
	}
	for _, r := range resp.Msg.Results {
		if r.GetId() == "draft" {
			t.Fatal("unreviewed asset leaked")
		}
		if r.GetSnippet() == "image-tools://secret-output" {
			t.Fatal("storage reference leaked")
		}
	}
}
