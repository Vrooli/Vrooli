package signals

import "testing"

func collectUI(t *testing.T, root string) UISignals {
	t.Helper()
	snap := Snapshot{Root: root}
	if err := (uiCollector{}).Collect(&snap); err != nil {
		t.Fatal(err)
	}
	return snap.UI
}

func TestUINoSourcesNotCollected(t *testing.T) {
	sig := collectUI(t, t.TempDir())
	if sig.Collected {
		t.Fatal("no ui/ dir must report Collected=false")
	}
}

func TestUITemplateDetection(t *testing.T) {
	tests := []struct {
		name         string
		app          string
		wantTemplate bool
	}{
		{
			name:         "starter signature",
			app:          "// This starter UI is intentionally minimal\nexport const App = () => null;\n",
			wantTemplate: true,
		},
		{
			name:         "placeholder signature",
			app:          "const x = 'TEMPLATE_PLACEHOLDER';\nexport const App = () => null;\n",
			wantTemplate: true,
		},
		{
			name:         "small file self-describing as minimal",
			app:          "// keep this minimal for now\nexport const App = () => null;\n",
			wantTemplate: true,
		},
		{
			name:         "custom app",
			app:          "import { useState } from 'react';\nexport const App = () => { const [n] = useState(0); return n; };\n",
			wantTemplate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "ui/src/App.tsx", tt.app)

			sig := collectUI(t, root)
			if !sig.Collected {
				t.Fatal("want Collected=true")
			}
			if sig.IsTemplate != tt.wantTemplate {
				t.Fatalf("IsTemplate = %v, want %v", sig.IsTemplate, tt.wantTemplate)
			}
		})
	}
}

func TestUIEndpointExtraction(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "ui/src/App.tsx", `
export const App = () => {
	fetch('/api/v1/tasks');
	fetch('/api/v1/tasks'); // duplicate: counted once
	axios.get('/api/v1/users');
	buildApiUrl('/api/v1/search');
	const h = '/health';
	return null;
};
`)
	// Files under skip dirs must not contribute.
	writeFile(t, root, "ui/src/node_modules/dep/index.js", `fetch('/api/v1/hidden');`)

	sig := collectUI(t, root)
	if sig.APIEndpoints != 4 {
		t.Fatalf("APIEndpoints = %d, want 4 unique", sig.APIEndpoints)
	}
	if sig.APIBeyondHealth != 3 {
		t.Fatalf("APIBeyondHealth = %d, want 3 (excludes /health)", sig.APIBeyondHealth)
	}
}

func TestUIRoutingDetection(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		wantHas    bool
		wantRoutes int
	}{
		{
			name: "react-router in entry",
			files: map[string]string{
				"ui/src/App.tsx": `import { Routes, Route } from 'react-router-dom';
export const App = () => (<Routes><Route path="/" /><Route path="/about" /></Routes>);`,
			},
			wantHas: true, wantRoutes: 2,
		},
		{
			name: "state-based views",
			files: map[string]string{
				"ui/src/App.tsx": `type View = 'home';
const views = ['home', 'settings', 'reports'];
export const App = () => null;`,
			},
			wantHas: true, wantRoutes: 4,
		},
		{
			name: "lazy loaded pages",
			files: map[string]string{
				"ui/src/App.tsx": `import { lazy } from 'react';
const Home = lazy(() => import('./Home'));
const About = lazy(() => import('./About'));
export const App = () => null;`,
			},
			wantHas: true, wantRoutes: 2,
		},
		{
			name: "routes file fallback",
			files: map[string]string{
				"ui/src/main.tsx":   `export const main = () => null;`,
				"ui/src/routes.tsx": `import { Route } from 'react-router-dom';\nexport const routes = (<><Route path="/" /></>);`,
			},
			wantHas: true, wantRoutes: 1,
		},
		{
			name: "no routing",
			files: map[string]string{
				"ui/src/App.tsx": `import { useState } from 'react';\nexport const App = () => useState(0)[0];`,
			},
			wantHas: false, wantRoutes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for rel, content := range tt.files {
				writeFile(t, root, rel, content)
			}

			sig := collectUI(t, root)
			if sig.HasRouting != tt.wantHas || sig.RouteCount != tt.wantRoutes {
				t.Fatalf("routing = %v/%d, want %v/%d",
					sig.HasRouting, sig.RouteCount, tt.wantHas, tt.wantRoutes)
			}
		})
	}
}

func TestUICountsAndLOC(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "ui/src/App.tsx", "line1\nline2\nline3\n")
	writeFile(t, root, "ui/src/components/Button.tsx", "a\nb\n")
	writeFile(t, root, "ui/src/components/Card.vue", "a\n")
	writeFile(t, root, "ui/src/pages/Home.tsx", "a\n")
	writeFile(t, root, "ui/src/styles.css", "ignored\n")
	writeFile(t, root, "ui/src/dist/bundle.js", "ignored\n")

	sig := collectUI(t, root)
	if sig.FileCount != 4 {
		t.Fatalf("FileCount = %d, want 4 source files", sig.FileCount)
	}
	if sig.ComponentCount != 2 {
		t.Fatalf("ComponentCount = %d, want 2", sig.ComponentCount)
	}
	if sig.PageCount != 1 {
		t.Fatalf("PageCount = %d, want 1", sig.PageCount)
	}
	if sig.TotalLOC != 7 {
		t.Fatalf("TotalLOC = %d, want 7", sig.TotalLOC)
	}
}

func TestUIFlatLayoutFallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "ui/App.tsx", "export const App = () => fetch('/api/v1/x');\n")

	sig := collectUI(t, root)
	if !sig.Collected || sig.FileCount != 1 || sig.APIEndpoints != 1 {
		t.Fatalf("ui = %+v, want flat ui/ collected", sig)
	}
}
