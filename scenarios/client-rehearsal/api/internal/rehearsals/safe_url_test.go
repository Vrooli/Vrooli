package rehearsals

import (
	"encoding/json"
	"testing"
)

func TestPublic20047HTTPURLExpressionsResolveOnlyDeclaredSyntheticNodeValues(t *testing.T) {
	var document struct {
		Nodes []struct {
			ID          string         `json:"id"`
			Name        string         `json:"name"`
			Type        string         `json:"type"`
			TypeVersion float64        `json:"typeVersion"`
			Parameters  map[string]any `json:"parameters"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(readFixture(t, "public-workflows/20047.json")), &document); err != nil {
		t.Fatal(err)
	}
	var settings WorkflowNode
	for _, source := range document.Nodes {
		if source.Name == "Set Static Content Settings" {
			node, err := normalizeWorkflowNodeVersion(source.ID, source.Name, source.Type, source.Parameters, source.TypeVersion)
			if err != nil {
				t.Fatalf("normalize static synthetic settings: %v", err)
			}
			settings = node
		}
	}
	if settings.SetValues == nil {
		t.Fatal("public static settings node did not yield bounded values")
	}
	settings.SetValues["pineconeIndexHost"] = "pinecone.synthetic.invalid"
	prior := map[string][]map[string]any{settings.Name: {settings.SetValues}}
	resolved := map[string]string{}
	for _, source := range document.Nodes {
		if source.Type != "n8n-nodes-base.httpRequest" {
			continue
		}
		node, err := normalizeWorkflowNodeVersion(source.ID, source.Name, source.Type, source.Parameters, source.TypeVersion)
		if err != nil {
			t.Fatalf("normalize %s: %v", source.Name, err)
		}
		request, err := requestWithSafeURLExpression(*node.Request, node.RequestURLExpression, prior)
		if err != nil {
			t.Fatalf("resolve %s: %v", source.Name, err)
		}
		if request.Host != "wordpress.synthetic.invalid" && request.Host != "pinecone.synthetic.invalid" {
			t.Fatalf("%s resolved to an undeclared host %q", source.Name, request.Host)
		}
		resolved[source.Name] = request.Method + " " + request.Host + request.URL
	}
	want := map[string]string{
		"Fetch Static Pages":        "GET wordpress.synthetic.invalid/wp-json/wp/v2/pages",
		"Fetch Blog Posts":          "GET wordpress.synthetic.invalid/wp-json/wp/v2/posts",
		"Remove Old Content Chunks": "POST pinecone.synthetic.invalid/vectors/delete",
		"Fetch Full Batch Pages":    "GET wordpress.synthetic.invalid/wp-json/wp/v2/pages",
		"Fetch Full Batch Posts":    "GET wordpress.synthetic.invalid/wp-json/wp/v2/posts",
		"Remove Deleted Vectors":    "POST pinecone.synthetic.invalid/vectors/delete",
	}
	if len(resolved) != len(want) {
		t.Fatalf("resolved URL count mismatch: got %d want %d (%#v)", len(resolved), len(want), resolved)
	}
	for name, expected := range want {
		if resolved[name] != expected {
			t.Errorf("%s URL = %q, want %q", name, resolved[name], expected)
		}
	}
}

func TestSafeSyntheticURLExpressionRejectsCodeAndRealDestinations(t *testing.T) {
	for _, expression := range []string{
		`={{ fetch('https://example.com') }}`,
		`={{ $('Settings').first().json.url.replace(/.*/, 'elsewhere') }}`,
	} {
		if _, err := parseSafeSyntheticURLExpression(expression); err == nil {
			t.Errorf("unsafe URL expression must be rejected: %s", expression)
		}
	}
	prior := map[string][]map[string]any{"Settings": {{"url": "https://example.com"}}}
	if _, err := resolveSafeSyntheticURLExpression(`={{ $('Settings').first().json.url }}`, prior); err == nil {
		t.Fatal("resolved real host must be rejected")
	}
}

func TestSafeSyntheticURLTemplateResolvesOnlyCurrentSyntheticURLs(t *testing.T) {
	template := `=https://{{$json["body"]["url"]}}`
	current := map[string]any{"body": map[string]any{"url": "article.synthetic.invalid/story"}}
	request, err := requestWithSafeURLExpression(Step{Method: "GET"}, template, nil, current)
	if err != nil {
		t.Fatalf("resolve bounded current-item URL: %v", err)
	}
	if request.Host != "article.synthetic.invalid" || request.URL != "/story" {
		t.Fatalf("unexpected resolved synthetic destination: host=%q path=%q", request.Host, request.URL)
	}
	for _, value := range []string{
		"example.com/path",
		"user@article.synthetic.invalid/story",
		"article.synthetic.invalid/story?token=secret",
		"article.synthetic.invalid/story#fragment",
		"article.synthetic.invalid/../private",
		"article.synthetic.invalid/%2e%2e/private",
	} {
		t.Run(value, func(t *testing.T) {
			input := map[string]any{"body": map[string]any{"url": value}}
			if _, err := requestWithSafeURLExpression(Step{Method: "GET"}, template, nil, input); err == nil {
				t.Fatalf("accepted unsafe dynamic URL %q", value)
			}
		})
	}
}

func TestSafeSyntheticURLTemplateRejectsUnboundedExpressions(t *testing.T) {
	for _, raw := range []string{
		`=https://{{$node["Other"].json.url}}`,
		`=https://{{$json["body"]["url"]}}/path?query={{$json.path}}`,
		"=https://{{$json.url}}\r",
	} {
		if _, err := parseSafeSyntheticURLExpression(raw); err == nil {
			t.Errorf("accepted unsupported URL template %q", raw)
		}
	}
}
