package checks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ui-health/internal/uiinterop"
)

func TestShellOwnership(t *testing.T) {
	const libraryImport = `import {AppShell as LibraryShell} from '@vrooli/react-component-library/AppShell/2';`
	const good = libraryImport + `export function AppShell(){return <LibraryShell><main><header>Results</header></main></LibraryShell>}`
	cases := []struct {
		name, entry, extra, ejection string
		pass                         bool
		message                      string
	}{
		{name: "library mount with page header", entry: good, pass: true},
		{name: "direct library reexport", entry: `export {AppShell} from '@vrooli/react-component-library/AppShell/2'`, pass: true},
		{name: "type-only library reexport", entry: `export type {AppShell} from '@vrooli/react-component-library/AppShell/2'`, message: "export does not exist"},
		{name: "type-only export specifier", entry: `export {type AppShell} from '@vrooli/react-component-library/AppShell/2'`, message: "export does not exist"},
		{name: "unused import", entry: libraryImport + `export function AppShell(){return <main/>}`, message: "does not render"},
		{name: "comment is not mount", entry: `// <LibraryShell/>
export function AppShell(){return <main/>}`, message: "does not render"},
		{name: "unused helper is not mount", entry: libraryImport + `function Unused(){return <LibraryShell/>} export function AppShell(){return <main/>}`, message: "does not render"},
		{name: "unrelated reference is not mount", entry: libraryImport + `export function AppShell(){const ignored=LibraryShell;return <main/>}`, message: "does not render"},
		{name: "unused nested function is not mount", entry: libraryImport + `export function AppShell(){function Unused(){return <LibraryShell/>} return <main/>}`, message: "does not render"},
		{name: "unused nested arrow is not mount", entry: libraryImport + `export function AppShell(){const Unused=()=> <LibraryShell/>; return <main/>}`, message: "does not render"},
		{name: "utility header is chrome", entry: libraryImport + `export function AppShell(){return <LibraryShell><header><button>Theme</button></header></LibraryShell>}`, message: "application header"},
		{name: "modal backdrop allowed", entry: good, extra: `export const Modal=()=> <div className="fixed inset-0"><div role="dialog">Confirm</div></div>`, pass: true},
		{name: "decorative child frame allowed", entry: good, extra: `export const Scene=()=> <div aria-hidden="true"><div className="fixed inset-0"/></div>`, pass: true},
		{name: "page bottom navigation competes with shell", entry: good, extra: `import {BottomNav as Tabs} from '@vrooli/react-component-library/BottomNav/1'; export const Page=()=> <section><Tabs items={[]}/></section>`, message: "application navigation"},
		{name: "chrome outside layout", entry: good, extra: `export const Toolbar=()=> <nav><a href="/runs">Runs</a></nav>`, message: "application navigation"},
		{name: "application header", entry: good, extra: `export const Toolbar=()=> <header><a href="/">Brand</a></header>`, message: "application header"},
		{name: "board frame", entry: good, extra: `export const Board=()=> <div className="fixed inset-0"/>`, message: "board frame"},
		{name: "alert fallback allowed", entry: good, extra: `export const Failure=()=> <div role="alert" className="min-h-dvh"><button>Retry</button></div>`, pass: true},
		{name: "alert cannot hide navigation", entry: good, extra: `export const Failure=()=> <nav role="alert" className="min-h-dvh"><a href="/">Home</a></nav>`, message: "board frame"},
		{name: "overlay allowed", entry: good, extra: `export const Modal=()=> <div role="dialog" className="fixed inset-0"/>`, pass: true},
		{name: "string is not chrome", entry: good, extra: "export const sample = '<nav><a href=\"/\">Home</a></nav>'", pass: true},
		{name: "scoped ejection", message: "Fixture: library cannot carry required interaction", entry: `export function AppShell(){return <nav><a href="/">Home</a></nav>}`, ejection: `{"archetype":"navigated-console","reason":"Fixture: library cannot carry required interaction","files":["ui/src/layout/AppShell.tsx"]}`, pass: true},
		{name: "ejection cannot cover other files", entry: `export function AppShell(){return <main/>}`, extra: `export const Board=()=> <div className="h-screen"/>`, ejection: `{"archetype":"navigated-console","reason":"Fixture gap","files":["ui/src/layout/AppShell.tsx"]}`, message: "board frame"},
		{name: "reason required", entry: good, ejection: `{"archetype":"navigated-console","files":["ui/src/layout/AppShell.tsx"]}`, message: "requires a reason"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{"ui/src/layout/AppShell.tsx": tc.entry}
			if tc.extra != "" {
				files["ui/src/features/board/Toolbar.tsx"] = tc.extra
			}
			ctx := uiinterop.CheckContext{ScenarioRoot: root, TechStack: []string{"React"}}
			for file, content := range files {
				writeShellFixture(t, root, file, content)
				ctx.Sources = append(ctx.Sources, uiinterop.SourceFile{RelPath: file, Content: content})
			}
			writeShellFixture(t, root, "ui/manifest.json", `{"shell":{"archetype":"navigated-console","asset":"AppShell","entry":"ui/src/layout/AppShell.tsx","export":"AppShell"}}`)
			if tc.ejection != "" {
				writeShellFixture(t, root, "docs/reference/component-library-gaps.md", "```shell-ejection\n"+tc.ejection+"\n```\n")
			}
			result := checkShellOwnership(ctx)
			encoded, _ := json.Marshal(result)
			if result.Passed != tc.pass {
				t.Fatalf("passed=%v, want %v: %s", result.Passed, tc.pass, encoded)
			}
			if tc.message != "" && !strings.Contains(string(encoded), tc.message) {
				t.Fatalf("missing %q: %s", tc.message, encoded)
			}
			if tc.extra != "" && !tc.pass && tc.message != "" && len(result.Violations) > 0 && result.Violations[0].FilePath != "ui/src/features/board/Toolbar.tsx" {
				t.Fatalf("wrong source attribution: %s", encoded)
			}
		})
	}
}

func TestShellOwnershipFollowsExportedWrapper(t *testing.T) {
	ctx := uiinterop.CheckContext{ScenarioRoot: t.TempDir(), Sources: []uiinterop.SourceFile{
		{RelPath: "ui/src/layout/AppShell.tsx", Content: `export {Console as AppShell} from './Console'`},
		{RelPath: "ui/src/layout/Console.tsx", Content: `import * as Shell from '@vrooli/react-component-library/AppShell/2'; export function Console(){return <Shell.AppShell/>}`},
	}}
	result, err := analyzeShellOwnership(ctx, shellDeclaration{Archetype: "navigated-console", Asset: "AppShell", Entry: "ui/src/layout/AppShell.tsx", Export: "AppShell"})
	if err != nil || !result.Mounted || len(result.Findings) != 0 {
		t.Fatalf("wrapper mount lost: %+v, %v", result, err)
	}
}

func TestShellOwnershipRequiresDeclaration(t *testing.T) {
	root := t.TempDir()
	writeShellFixture(t, root, "ui/manifest.json", `{"contract":{}}`)
	result := checkShellOwnership(uiinterop.CheckContext{ScenarioRoot: root})
	if result.Passed || len(result.Violations) != 1 || result.Violations[0].FilePath != "ui/manifest.json" {
		t.Fatalf("missing declaration passed: %+v", result)
	}
}

func writeShellFixture(t *testing.T, root, file, content string) {
	t.Helper()
	target := filepath.Join(root, file)
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestShellOwnershipDetectsPublicBrowserChrome(t *testing.T) {
	root := t.TempDir()
	writeShellFixture(t, root, "ui/src/layout/AppShell.tsx", `import {AppShell as LibraryShell} from '@vrooli/react-component-library/AppShell/2'; export function AppShell(){return <LibraryShell/>}`)
	writeShellFixture(t, root, "ui/public/navigation.jsx", `export function Navigation(){return <nav><a href="/runs">Runs</a></nav>}`)
	ctx := uiinterop.CheckContext{ScenarioRoot: root, Sources: uiinterop.WalkUISource(root, "ui")}
	result, err := analyzeShellOwnership(ctx, shellDeclaration{Archetype: "navigated-console", Asset: "AppShell", Entry: "ui/src/layout/AppShell.tsx", Export: "AppShell"})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range result.Findings {
		if finding.File == "ui/public/navigation.jsx" && strings.Contains(finding.Reason, "application navigation") {
			return
		}
	}
	t.Fatalf("public browser navigation escaped ownership check: %+v", result)
}

func TestShellOwnershipPublicEjectionIsFileScoped(t *testing.T) {
	root := t.TempDir()
	writeShellFixture(t, root, "ui/manifest.json", `{"shell":{"archetype":"navigated-console","asset":"AppShell","entry":"ui/src/layout/AppShell.tsx","export":"AppShell"}}`)
	writeShellFixture(t, root, "ui/src/layout/AppShell.tsx", `import {AppShell as LibraryShell} from '@vrooli/react-component-library/AppShell/2'; export function AppShell(){return <LibraryShell/>}`)
	writeShellFixture(t, root, "ui/public/navigation.jsx", `export function Navigation(){return <nav><a href="/runs">Runs</a></nav>}`)
	writeShellFixture(t, root, "docs/reference/component-library-gaps.md", "```shell-ejection\n"+`{"archetype":"navigated-console","reason":"Fixture: embedded browser navigation requires unsupported interaction","files":["ui/public/navigation.jsx"]}`+"\n```\n")
	check := func() uiinterop.RuleResult {
		return checkShellOwnership(uiinterop.CheckContext{ScenarioRoot: root, Sources: uiinterop.WalkUISource(root, "ui")})
	}
	if result := check(); !result.Passed {
		t.Fatalf("documented public exception rejected: %+v", result)
	}
	writeShellFixture(t, root, "ui/public/other.jsx", `export function Other(){return <nav><a href="/settings">Settings</a></nav>}`)
	result := check()
	if result.Passed || len(result.Violations) != 1 || result.Violations[0].FilePath != "ui/public/other.jsx" {
		t.Fatalf("exception escaped its exact file: %+v", result)
	}
}

func TestShellOwnershipDetectsRootBrowserEntryChrome(t *testing.T) {
	root := t.TempDir()
	writeShellFixture(t, root, "ui/src/layout/AppShell.tsx", `import {AppShell as LibraryShell} from '@vrooli/react-component-library/AppShell/2'; export function AppShell(){return <LibraryShell/>}`)
	writeShellFixture(t, root, "ui/app.js", `export function Navigation(){return <nav><a href="/runs">Runs</a></nav>}`)
	result, err := analyzeShellOwnership(uiinterop.CheckContext{ScenarioRoot: root, Sources: uiinterop.WalkUISource(root, "ui")}, shellDeclaration{Archetype: "navigated-console", Asset: "AppShell", Entry: "ui/src/layout/AppShell.tsx", Export: "AppShell"})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range result.Findings {
		if finding.File == "ui/app.js" && strings.Contains(finding.Reason, "application navigation") {
			return
		}
	}
	t.Fatalf("root browser entry escaped ownership analysis: %+v", result)
}

func TestShellOwnershipStaticHTML(t *testing.T) {
	cases := []struct {
		name, markup string
		violation    bool
	}{
		{"application header", `<header><a href="/">Product</a><button>Theme</button></header><main></main>`, true},
		{"banner role", `<div role="banner">Product</div><main></main>`, true},
		{"article heading", `<article><header><h1>Release notes</h1></header></article>`, false},
		{"section heading", `<section><header><h2>Recent runs</h2></header></section>`, false},
		{"explicit banner within main", `<main><div role="banner">Product</div></main>`, true},
		{"page heading", `<main><header><h1>Run detail</h1></header></main>`, false},
		{"dialog heading", `<dialog><header><h2>Confirm</h2></header></dialog>`, false},
		{"root navigation", `<nav><a href="/runs">Runs</a></nav><main id="root"></main>`, true},
		{"public navigation role", `<div role="navigation"><a href="/">Home</a></div>`, true},
		{"comment and script examples", `<!-- <nav><a href="/">Home</a></nav> --><script>const example='<nav>Example</nav>'</script><div id="root"></div>`, false},
		{"inert template", `<template><nav><a href="/">Home</a></nav></template><div id="root"></div>`, false},
		{"application navigation under main", `<main><nav><a href="/runs">Runs</a></nav></main>`, true},
		{"relative application navigation under main", `<main><nav><a href="settings.html">Settings</a></nav></main>`, true},
		{"inert link does not change section nav", `<main><nav><a href="#details">Details</a><template><a href="/runs">Runs</a></template></nav></main>`, false},
		{"content section navigation", `<main><nav><a href="#details">Details</a></nav></main>`, false},
		{"dialog navigation", `<dialog open><nav><a href="/help">Help</a></nav></dialog>`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeShellFixture(t, root, "ui/src/layout/AppShell.tsx", `import {AppShell as LibraryShell} from '@vrooli/react-component-library/AppShell/2'; export function AppShell(){return <LibraryShell/>}`)
			htmlPath := "ui/index.html"
			if tc.name == "public navigation role" {
				htmlPath = "ui/public/index.html"
			}
			writeShellFixture(t, root, htmlPath, tc.markup)
			result, err := analyzeShellOwnership(uiinterop.CheckContext{ScenarioRoot: root, Sources: uiinterop.WalkUISource(root, "ui")}, shellDeclaration{Archetype: "navigated-console", Asset: "AppShell", Entry: "ui/src/layout/AppShell.tsx", Export: "AppShell"})
			if err != nil {
				t.Fatal(err)
			}
			if (len(result.Findings) > 0) != tc.violation {
				t.Fatalf("HTML ownership mismatch: %+v", result)
			}
		})
	}
}
