package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"app-monitor-api/repository"
)

// =============================================================================
// Helper: create directories and write file under tmpDir
// =============================================================================

func writeTestFile(t *testing.T, base string, relPath string, content string) {
	t.Helper()
	full := filepath.Join(base, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("MkdirAll for %s: %v", relPath, err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile for %s: %v", relPath, err)
	}
}

// =============================================================================
// 1. checkApiBaseDep
// =============================================================================

func TestCheckApiBaseDep_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/package.json", `{
		"dependencies": {
			"@vrooli/api-base": "^1.0.0",
			"react": "^18.0.0"
		}
	}`)

	r := checkApiBaseDep(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_api_base_dep" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckApiBaseDep_PassDevDependencies(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/package.json", `{
		"devDependencies": {
			"@vrooli/api-base": "workspace:*"
		}
	}`)

	r := checkApiBaseDep(tmp)
	if !r.Passed {
		t.Errorf("expected pass via devDependencies, got fail: %s", r.Message)
	}
}

func TestCheckApiBaseDep_Fail(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/package.json", `{
		"dependencies": {
			"react": "^18.0.0"
		}
	}`)

	r := checkApiBaseDep(tmp)
	if r.Passed {
		t.Error("expected fail when @vrooli/api-base is missing")
	}
	if !strings.Contains(r.Message, "@vrooli/api-base not found") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckApiBaseDep_NoFile(t *testing.T) {
	tmp := t.TempDir()
	// no ui/package.json at all

	r := checkApiBaseDep(tmp)
	if r.Passed {
		t.Error("expected fail when ui/package.json missing")
	}
	if !strings.Contains(r.Message, "cannot read") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

// =============================================================================
// 2. checkIframeBridgeDep
// =============================================================================

func TestCheckIframeBridgeDep_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/package.json", `{
		"dependencies": {
			"@vrooli/iframe-bridge": "^2.0.0"
		}
	}`)

	r := checkIframeBridgeDep(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_iframe_bridge_dep" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckIframeBridgeDep_Fail(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/package.json", `{
		"dependencies": {
			"react": "^18.0.0"
		}
	}`)

	r := checkIframeBridgeDep(tmp)
	if r.Passed {
		t.Error("expected fail when @vrooli/iframe-bridge is missing")
	}
}

// =============================================================================
// 3. checkHardcodedLocalhost
// =============================================================================

func TestCheckHardcodedLocalhost_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/App.tsx", `import React from 'react';
export default function App() {
	const api = resolveApiBase();
	return <div>Hello</div>;
}
`)

	r := checkHardcodedLocalhost(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_hardcoded_localhost" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckHardcodedLocalhost_Fail(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/api/client.ts", `
const BASE_URL = "http://localhost:3000/api";
export const fetch = () => {};
`)

	r := checkHardcodedLocalhost(tmp)
	if r.Passed {
		t.Error("expected fail when localhost:PORT is present")
	}
	if !strings.Contains(r.Message, "hardcoded localhost") {
		t.Errorf("unexpected message: %s", r.Message)
	}
	if r.FilePath == "" {
		t.Error("expected FilePath to be set")
	}
	if r.Line == 0 {
		t.Error("expected Line to be set")
	}
}

func TestCheckHardcodedLocalhost_CommentIgnored(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/App.tsx", `
// This used to be localhost:3000
/* localhost:8080 is the old URL */
export default function App() { return null; }
`)

	r := checkHardcodedLocalhost(tmp)
	if !r.Passed {
		t.Errorf("expected pass when localhost is only in comments, got fail: %s", r.Message)
	}
}

func TestCheckHardcodedLocalhost_TestFileIgnored(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/lib/api.test.ts", `
const BASE = "http://localhost:3000/api/v1";
`)
	writeTestFile(t, tmp, "ui/src/App.tsx", `export default function App() { return null; }`)

	r := checkHardcodedLocalhost(tmp)
	if !r.Passed {
		t.Errorf("expected pass when localhost is only in test file, got fail: %s", r.Message)
	}
}

func TestCheckHardcodedLocalhost_NoSrcDir(t *testing.T) {
	tmp := t.TempDir()
	// no ui/src/ directory

	r := checkHardcodedLocalhost(tmp)
	if !r.Passed {
		t.Errorf("expected pass when ui/src/ doesn't exist, got: %s", r.Message)
	}
}

// =============================================================================
// 4. checkRelativeBase
// =============================================================================

func TestCheckRelativeBase_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/vite.config.ts", `
import { defineConfig } from 'vite';
// INTEROP-CRITICAL: base must be relative for iframe embedding
export default defineConfig({
  base: './',
  plugins: [],
});
`)

	r := checkRelativeBase(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_relative_base" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
	if r.Line == 0 {
		t.Error("expected Line to be set")
	}
}

func TestCheckRelativeBase_FailWrongBase(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/vite.config.ts", `
export default defineConfig({
  base: '/my-app/',
  plugins: [],
});
`)

	r := checkRelativeBase(tmp)
	if r.Passed {
		t.Error("expected fail when base is not './'")
	}
	if !strings.Contains(r.Message, "not to './'") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckRelativeBase_FailNoBase(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/vite.config.ts", `
export default defineConfig({
  plugins: [],
});
`)

	r := checkRelativeBase(tmp)
	if r.Passed {
		t.Error("expected fail when no base config is set")
	}
	if !strings.Contains(r.Message, "no base config") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckRelativeBase_FailNoFile(t *testing.T) {
	tmp := t.TempDir()
	// No vite config files at all

	r := checkRelativeBase(tmp)
	if r.Passed {
		t.Error("expected fail when vite config is missing")
	}
	if !strings.Contains(r.Message, "cannot read") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckRelativeBase_JsFallback(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/vite.config.js", `
export default defineConfig({
  base: './',
});
`)

	r := checkRelativeBase(tmp)
	if !r.Passed {
		t.Errorf("expected pass via vite.config.js fallback, got fail: %s", r.Message)
	}
	if r.FilePath != "ui/vite.config.js" {
		t.Errorf("expected FilePath to be ui/vite.config.js, got %s", r.FilePath)
	}
}

// =============================================================================
// 5. checkRouterBasename
// =============================================================================

func TestCheckRouterBasename_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/App.tsx", `
import { BrowserRouter } from 'react-router-dom';
export default function App() {
  return <BrowserRouter basename="/app">
    <Routes />
  </BrowserRouter>;
}
`)

	r := checkRouterBasename(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_router_basename" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckRouterBasename_Fail(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/App.tsx", `
import { BrowserRouter } from 'react-router-dom';
export default function App() {
  return <BrowserRouter>
    <Routes />
  </BrowserRouter>;
}
`)

	r := checkRouterBasename(tmp)
	if r.Passed {
		t.Error("expected fail when BrowserRouter has no basename")
	}
	if !strings.Contains(r.Message, "missing basename") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckRouterBasename_Skip_NoBrowserRouter(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/App.tsx", `
import { HashRouter } from 'react-router-dom';
export default function App() {
  return <HashRouter><Routes /></HashRouter>;
}
`)

	r := checkRouterBasename(tmp)
	if !r.Skipped {
		t.Error("expected skip when no BrowserRouter found")
	}
	if r.SkipReason == "" {
		t.Error("expected SkipReason to be set")
	}
}

func TestCheckRouterBasename_Skip_NoSrcDir(t *testing.T) {
	tmp := t.TempDir()
	// no ui/src/ directory at all

	r := checkRouterBasename(tmp)
	if !r.Skipped {
		t.Error("expected skip when no ui/src/ directory")
	}
}

// =============================================================================
// 6. checkNoCustomServer
// =============================================================================

func TestCheckNoCustomServer_Pass_NoServerFile(t *testing.T) {
	tmp := t.TempDir()
	// No server.js at all
	writeTestFile(t, tmp, "ui/package.json", `{}`)

	r := checkNoCustomServer(tmp)
	if !r.Passed {
		t.Errorf("expected pass when no server file exists, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_no_custom_server" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckNoCustomServer_Pass_StandardServer(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/server.js", `
import { startScenarioServer } from '@vrooli/scenario-server';
startScenarioServer();
`)

	r := checkNoCustomServer(tmp)
	if !r.Passed {
		t.Errorf("expected pass for standard server, got fail: %s", r.Message)
	}
}

func TestCheckNoCustomServer_Fail_Express(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/server.js", `
const app = express();
app.use(express.static('dist'));
app.listen(3000);
`)

	r := checkNoCustomServer(tmp)
	if r.Passed {
		t.Error("expected fail when express() is found")
	}
	if !strings.Contains(r.Message, "custom server code") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckNoCustomServer_Fail_CreateServer(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/server.ts", `
import http from 'http';
const server = http.createServer(handler);
server.listen(3000);
`)

	r := checkNoCustomServer(tmp)
	if r.Passed {
		t.Error("expected fail when createServer is found")
	}
}

// =============================================================================
// 7. checkBridgeInit
// =============================================================================

func TestCheckBridgeInit_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
initIframeBridgeChild({ appId: 'my-app' });
ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
`)

	r := checkBridgeInit(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_bridge_init" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
	if r.Line == 0 {
		t.Error("expected Line to be set")
	}
}

func TestCheckBridgeInit_Fail(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import ReactDOM from 'react-dom/client';
ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
`)

	r := checkBridgeInit(tmp)
	if r.Passed {
		t.Error("expected fail when initIframeBridgeChild is missing")
	}
	if !strings.Contains(r.Message, "initIframeBridgeChild not found") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckBridgeInit_FailNoEntryFile(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/other.tsx", `export default function Foo() {}`)

	r := checkBridgeInit(tmp)
	if r.Passed {
		t.Error("expected fail when no entry file exists")
	}
	if !strings.Contains(r.Message, "no main entry file") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

// =============================================================================
// 8. checkResolveApiBaseSingle
// =============================================================================

func TestCheckResolveApiBaseSingle_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/api/client.ts", `
import { resolveApiBase } from '@vrooli/api-base';
const BASE = resolveApiBase();
`)
	writeTestFile(t, tmp, "ui/src/App.tsx", `
export default function App() { return null; }
`)

	r := checkResolveApiBaseSingle(tmp)
	if !r.Passed {
		t.Errorf("expected pass with exactly 1 file, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_resolve_api_base_single" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckResolveApiBaseSingle_FailZero(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/App.tsx", `export default function App() { return null; }`)

	r := checkResolveApiBaseSingle(tmp)
	if r.Passed {
		t.Error("expected fail when resolveApiBase is not found in any file")
	}
	if !strings.Contains(r.Message, "not found") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckResolveApiBaseSingle_PassTwo(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/api/client.ts", `
import { resolveApiBase } from '@vrooli/api-base';
const BASE = resolveApiBase();
`)
	writeTestFile(t, tmp, "ui/src/hooks/useStream.ts", `
import { resolveApiBase } from '@vrooli/api-base';
const SSE_BASE = resolveApiBase({ appendSuffix: false });
`)

	r := checkResolveApiBaseSingle(tmp)
	if !r.Passed {
		t.Errorf("expected pass when resolveApiBase in 2 files, got fail: %s", r.Message)
	}
}

func TestCheckResolveApiBaseSingle_FailThreeOrMore(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/api/client.ts", `
import { resolveApiBase } from '@vrooli/api-base';
const BASE = resolveApiBase();
`)
	writeTestFile(t, tmp, "ui/src/hooks/useStream.ts", `
import { resolveApiBase } from '@vrooli/api-base';
const SSE = resolveApiBase();
`)
	writeTestFile(t, tmp, "ui/src/other/service.ts", `
import { resolveApiBase } from '@vrooli/api-base';
const URL = resolveApiBase();
`)

	r := checkResolveApiBaseSingle(tmp)
	if r.Passed {
		t.Error("expected fail when resolveApiBase found in more than 2 files")
	}
	if !strings.Contains(r.Message, "3 files") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckResolveApiBaseSingle_SkipNoSrcDir(t *testing.T) {
	tmp := t.TempDir()
	// No ui/src/ at all

	r := checkResolveApiBaseSingle(tmp)
	if !r.Skipped {
		t.Error("expected skip when no ui/src/ directory")
	}
}

func TestCheckResolveApiBaseSingle_TestFileIgnored(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/lib/api.ts", `
import { resolveApiBase } from '@vrooli/api-base';
const BASE = resolveApiBase();
`)
	writeTestFile(t, tmp, "ui/src/lib/api.test.ts", `
import { resolveApiBase } from '@vrooli/api-base';
vi.mock('@vrooli/api-base', () => ({ resolveApiBase: vi.fn() }));
`)

	r := checkResolveApiBaseSingle(tmp)
	if !r.Passed {
		t.Errorf("expected pass when test file has resolveApiBase (should be excluded), got fail: %s", r.Message)
	}
	if !strings.Contains(r.Message, "1 file(s)") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

// =============================================================================
// 9. checkShortcutRelay
// =============================================================================

func TestCheckShortcutRelay_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/hooks/useKeyboard.ts", `
import { emitShortcutIntent } from '@vrooli/iframe-bridge';
export function useKeyboard() {
  // keydown handler that uses emitShortcutIntent
  document.addEventListener('keydown', (e) => {
    emitShortcutIntent(e);
  });
}
`)

	r := checkShortcutRelay(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_shortcut_relay" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckShortcutRelay_PassRelayInApp(t *testing.T) {
	tmp := t.TempDir()
	// Hook exists with keyboard handling but no emitShortcutIntent
	writeTestFile(t, tmp, "ui/src/hooks/useKeyboardShortcuts.ts", `
export function useKeyboardShortcuts(shortcuts, options) {
  const { onUnhandledShortcut } = options;
  document.addEventListener('keydown', handler);
}
`)
	// But App.tsx has the relay via callback pattern
	writeTestFile(t, tmp, "ui/src/App.tsx", `
import { emitShortcutIntent } from '@vrooli/iframe-bridge';
const handleUnhandled = (shortcut) => {
  emitShortcutIntent({ action: 'host.open-global-switcher', outcome: 'noop' });
};
`)

	r := checkShortcutRelay(tmp)
	if !r.Passed {
		t.Errorf("expected pass when emitShortcutIntent is in App.tsx callback, got fail: %s", r.Message)
	}
}

func TestCheckShortcutRelay_Fail(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/hooks/useKeyboard.ts", `
export function useKeyboard() {
  document.addEventListener('keydown', (e) => {
    console.log(e.key);
  });
}
`)

	r := checkShortcutRelay(tmp)
	if r.Passed {
		t.Error("expected fail when keyboard hooks exist without emitShortcutIntent")
	}
	if !strings.Contains(r.Message, "emitShortcutIntent not found") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckShortcutRelay_Skip_NoHooksDir(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/App.tsx", `export default function App() { return null; }`)

	r := checkShortcutRelay(tmp)
	if !r.Skipped {
		t.Error("expected skip when no hooks/ directory")
	}
	if r.SkipReason == "" {
		t.Error("expected SkipReason to be set")
	}
}

func TestCheckShortcutRelay_Skip_NoKeyboardHooks(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/hooks/useTheme.ts", `
export function useTheme() {
  return { theme: 'dark' };
}
`)

	r := checkShortcutRelay(tmp)
	if !r.Skipped {
		t.Error("expected skip when hooks exist but none are keyboard-related")
	}
}

// =============================================================================
// 10. checkNoScatteredKeydown
// =============================================================================

func TestCheckNoScatteredKeydown_Pass(t *testing.T) {
	tmp := t.TempDir()
	// Only in hooks/ which is allowed
	writeTestFile(t, tmp, "ui/src/hooks/useKeyboard.ts", `
document.addEventListener('keydown', handler);
`)
	writeTestFile(t, tmp, "ui/src/App.tsx", `
export default function App() { return null; }
`)

	r := checkNoScatteredKeydown(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_no_scattered_keydown" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckNoScatteredKeydown_PassInDialog(t *testing.T) {
	tmp := t.TempDir()
	// keydown in dialog component is allowed
	writeTestFile(t, tmp, "ui/src/components/ConfirmDialog.tsx", `
document.addEventListener('keydown', handler);
`)

	r := checkNoScatteredKeydown(tmp)
	if !r.Passed {
		t.Errorf("expected pass for dialog component, got fail: %s", r.Message)
	}
}

func TestCheckNoScatteredKeydown_PassInDropdown(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/components/ui/dropdown.tsx", `
document.addEventListener('keydown', handler);
`)

	r := checkNoScatteredKeydown(tmp)
	if !r.Passed {
		t.Errorf("expected pass for dropdown component, got fail: %s", r.Message)
	}
}

func TestCheckNoScatteredKeydown_PassInOverlay(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/components/AIMergeOverlay.tsx", `
document.addEventListener('keydown', handler);
`)

	r := checkNoScatteredKeydown(tmp)
	if !r.Passed {
		t.Errorf("expected pass for overlay component, got fail: %s", r.Message)
	}
}

func TestCheckNoScatteredKeydown_PassInPopup(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/components/SlashCommandPopup.tsx", `
document.addEventListener('keydown', handler);
`)

	r := checkNoScatteredKeydown(tmp)
	if !r.Passed {
		t.Errorf("expected pass for popup component, got fail: %s", r.Message)
	}
}

func TestCheckNoScatteredKeydown_PassInSelector(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/components/IconSelector.tsx", `
document.addEventListener('keydown', handler);
`)

	r := checkNoScatteredKeydown(tmp)
	if !r.Passed {
		t.Errorf("expected pass for selector component, got fail: %s", r.Message)
	}
}

func TestCheckNoScatteredKeydown_Fail(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/components/Sidebar.tsx", `
export function Sidebar() {
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeSidebar();
  });
}
`)

	r := checkNoScatteredKeydown(tmp)
	if r.Passed {
		t.Error("expected fail when keydown is scattered outside hooks/")
	}
	if !strings.Contains(r.Message, "keydown listeners found outside") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckNoScatteredKeydown_Skip_NoSrcDir(t *testing.T) {
	tmp := t.TempDir()
	// No ui/src/ directory at all

	r := checkNoScatteredKeydown(tmp)
	if !r.Skipped {
		t.Error("expected skip when no ui/src/ directory")
	}
}

// =============================================================================
// 11. checkBridgeAppId
// =============================================================================

func TestCheckBridgeAppId_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
initIframeBridgeChild({ appId: 'my-app' });
`)

	r := checkBridgeAppId(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_bridge_app_id" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckBridgeAppId_PassMultiLine(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
initIframeBridgeChild({
  appId: 'my-app',
  debug: true,
});
`)

	r := checkBridgeAppId(tmp)
	if !r.Passed {
		t.Errorf("expected pass for multi-line call, got fail: %s", r.Message)
	}
}

func TestCheckBridgeAppId_Fail(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
initIframeBridgeChild({ debug: true });
`)

	r := checkBridgeAppId(tmp)
	if r.Passed {
		t.Error("expected fail when appId is missing from initIframeBridgeChild call")
	}
	if !strings.Contains(r.Message, "missing appId") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckBridgeAppId_FailNoBridgeCall(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import ReactDOM from 'react-dom/client';
ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
`)

	r := checkBridgeAppId(tmp)
	if r.Passed {
		t.Error("expected fail when initIframeBridgeChild is not present")
	}
	if !strings.Contains(r.Message, "initIframeBridgeChild not found") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckBridgeAppId_FailNoEntryFile(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/other.tsx", `export default function Foo() {}`)

	r := checkBridgeAppId(tmp)
	if r.Passed {
		t.Error("expected fail when no main entry file exists")
	}
	if !strings.Contains(r.Message, "no main entry file") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

// =============================================================================
// 12. checkProtectiveComments
// =============================================================================

func TestCheckProtectiveComments_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/vite.config.ts", `
// INTEROP-CRITICAL: do not change base
export default defineConfig({ base: './' });
`)
	writeTestFile(t, tmp, "ui/src/main.tsx", `
// INTEROP-CRITICAL: bridge init must remain
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
initIframeBridgeChild({ appId: 'test' });
`)

	r := checkProtectiveComments(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_protective_comments" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckProtectiveComments_FailMissingVite(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/vite.config.ts", `
export default defineConfig({ base: './' });
`)
	writeTestFile(t, tmp, "ui/src/main.tsx", `
// INTEROP-CRITICAL: bridge init must remain
initIframeBridgeChild({ appId: 'test' });
`)

	r := checkProtectiveComments(tmp)
	if r.Passed {
		t.Error("expected fail when INTEROP-CRITICAL missing from vite config")
	}
	if !strings.Contains(r.Message, "ui/vite.config.ts") {
		t.Errorf("expected message to reference vite config, got: %s", r.Message)
	}
}

func TestCheckProtectiveComments_FailMissingMain(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/vite.config.ts", `
// INTEROP-CRITICAL: do not change base
export default defineConfig({ base: './' });
`)
	writeTestFile(t, tmp, "ui/src/main.tsx", `
initIframeBridgeChild({ appId: 'test' });
`)

	r := checkProtectiveComments(tmp)
	if r.Passed {
		t.Error("expected fail when INTEROP-CRITICAL missing from main.tsx")
	}
	if !strings.Contains(r.Message, "ui/src/main.tsx") {
		t.Errorf("expected message to reference main.tsx, got: %s", r.Message)
	}
}

func TestCheckProtectiveComments_FailBothMissing(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/vite.config.ts", `
export default defineConfig({ base: './' });
`)
	writeTestFile(t, tmp, "ui/src/main.tsx", `
initIframeBridgeChild({ appId: 'test' });
`)

	r := checkProtectiveComments(tmp)
	if r.Passed {
		t.Error("expected fail when both files lack INTEROP-CRITICAL")
	}
	if !strings.Contains(r.Message, "ui/vite.config.ts") || !strings.Contains(r.Message, "ui/src/main.tsx") {
		t.Errorf("expected message to reference both files, got: %s", r.Message)
	}
}

// =============================================================================
// Integration: CheckInteropCompliance
// =============================================================================

func TestCheckInteropCompliance_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a full compliant scenario structure
	writeTestFile(t, tmpDir, "ui/package.json", `{
		"dependencies": {
			"@vrooli/api-base": "^1.0.0",
			"@vrooli/iframe-bridge": "^2.0.0",
			"react": "^18.0.0"
		}
	}`)
	writeTestFile(t, tmpDir, "ui/vite.config.ts", `
// INTEROP-CRITICAL: base must be relative
export default defineConfig({ base: './' });
`)
	writeTestFile(t, tmpDir, "ui/src/main.tsx", `
// INTEROP-CRITICAL: bridge init required
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
initIframeBridgeChild({ appId: 'test-app' });
ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
`)
	writeTestFile(t, tmpDir, "ui/src/api/client.ts", `
import { resolveApiBase } from '@vrooli/api-base';
export const BASE = resolveApiBase();
`)
	writeTestFile(t, tmpDir, "ui/src/App.tsx", `
export default function App() { return <div>Hello</div>; }
`)

	mockRepo := &mockAppRepository{
		apps: []repository.App{
			{
				ID:           "test-app",
				Name:         "Test App",
				ScenarioName: "test-scenario",
				Path:         tmpDir,
			},
		},
	}
	service := NewAppService(mockRepo)

	report, err := service.CheckInteropCompliance(context.Background(), "test-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Scenario != "test-scenario" {
		t.Errorf("expected scenario 'test-scenario', got %q", report.Scenario)
	}

	if !report.HasUI {
		t.Error("expected HasUI to be true")
	}

	if report.TotalCount != 16 {
		t.Errorf("expected 16 total checks, got %d", report.TotalCount)
	}

	// Verify at least some checks passed
	if report.PassCount == 0 {
		t.Error("expected at least some checks to pass")
	}

	// Verify score is computed
	if report.Score < 0 || report.Score > 100 {
		t.Errorf("score out of range: %d", report.Score)
	}

	// Verify CheckedAt is set
	if report.CheckedAt.IsZero() {
		t.Error("expected CheckedAt to be set")
	}

	// Verify check IDs are set
	for _, check := range report.Checks {
		if check.CheckID == "" {
			t.Error("expected CheckID to be set on every check result")
		}
		if check.Severity == "" {
			t.Error("expected Severity to be set on every check result")
		}
	}
}

func TestCheckInteropCompliance_NoUIDir(t *testing.T) {
	tmpDir := t.TempDir()
	// No ui/ directory at all

	mockRepo := &mockAppRepository{
		apps: []repository.App{
			{
				ID:           "test-app",
				Name:         "Test App",
				ScenarioName: "test-scenario",
				Path:         tmpDir,
			},
		},
	}
	service := NewAppService(mockRepo)

	report, err := service.CheckInteropCompliance(context.Background(), "test-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.HasUI {
		t.Error("expected HasUI to be false when no ui/ directory")
	}
	if len(report.Warnings) == 0 {
		t.Error("expected warnings about missing ui/ directory")
	}
	if len(report.Checks) != 0 {
		t.Errorf("expected 0 checks when no ui/, got %d", len(report.Checks))
	}
}

func TestCheckInteropCompliance_EmptyPath(t *testing.T) {
	mockRepo := &mockAppRepository{
		apps: []repository.App{
			{
				ID:           "test-app",
				Name:         "Test App",
				ScenarioName: "test-scenario",
				Path:         "",
			},
		},
	}
	service := NewAppService(mockRepo)

	report, err := service.CheckInteropCompliance(context.Background(), "test-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.HasUI {
		t.Error("expected HasUI to be false when path is empty")
	}
	if len(report.Warnings) == 0 {
		t.Error("expected warnings about unknown path")
	}
}

func TestCheckInteropCompliance_EmptyAppID(t *testing.T) {
	service := NewAppService(nil)

	_, err := service.CheckInteropCompliance(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty app ID")
	}
}

func TestCheckInteropCompliance_AppNotFound(t *testing.T) {
	mockRepo := &mockAppRepository{apps: []repository.App{}}
	service := NewAppService(mockRepo)

	_, err := service.CheckInteropCompliance(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent app")
	}
}

// =============================================================================
// Integration: GetInteropStandards
// =============================================================================

func TestGetInteropStandards_Integration(t *testing.T) {
	// Build a parent dir that contains scenarios/test-scenario/ui/...
	parentDir := t.TempDir()
	scenarioRoot := filepath.Join(parentDir, "scenarios", "test-scenario")

	writeTestFile(t, scenarioRoot, "ui/package.json", `{
		"dependencies": {
			"react": "^18.0.0"
		}
	}`)
	writeTestFile(t, scenarioRoot, "ui/vite.config.ts", `
export default defineConfig({ base: '/' });
`)
	writeTestFile(t, scenarioRoot, "ui/src/main.tsx", `
ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
`)
	writeTestFile(t, scenarioRoot, "ui/src/App.tsx", `
export default function App() { return null; }
`)

	mockRepo := &mockAppRepository{apps: []repository.App{}}
	service := NewAppService(mockRepo)
	service.repoRoot = parentDir

	resp, err := service.GetInteropStandards(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.EntityName != "test-scenario" {
		t.Errorf("expected entity_name 'test-scenario', got %q", resp.EntityName)
	}

	// There should be violations since we omitted many required things
	if len(resp.Violations) == 0 {
		t.Error("expected violations for non-compliant scenario")
	}

	// Verify violation fields are populated
	for _, v := range resp.Violations {
		if v.RuleID == "" {
			t.Error("expected RuleID to be set on violation")
		}
		if v.Severity == "" {
			t.Error("expected Severity to be set on violation")
		}
		if v.Title == "" {
			t.Error("expected Title to be set on violation")
		}
		if v.Description == "" {
			t.Error("expected Description to be set on violation")
		}
		if v.Recommendation == "" {
			t.Errorf("expected Recommendation to be set for rule %s", v.RuleID)
		}
		if v.Metadata == nil {
			t.Errorf("expected Metadata to be set for rule %s", v.RuleID)
		}
		if _, ok := v.Metadata["slot"]; !ok {
			t.Errorf("expected 'slot' in metadata for rule %s", v.RuleID)
		}
	}
}

func TestGetInteropStandards_NoUI(t *testing.T) {
	parentDir := t.TempDir()
	scenarioRoot := filepath.Join(parentDir, "scenarios", "no-ui-scenario")
	if err := os.MkdirAll(scenarioRoot, 0755); err != nil {
		t.Fatalf("failed to create scenario root: %v", err)
	}
	// No ui/ directory

	mockRepo := &mockAppRepository{apps: []repository.App{}}
	service := NewAppService(mockRepo)
	service.repoRoot = parentDir

	resp, err := service.GetInteropStandards(context.Background(), "no-ui-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Violations) != 0 {
		t.Errorf("expected 0 violations when no UI exists, got %d", len(resp.Violations))
	}
}

func TestGetInteropStandards_EmptyScenarioName(t *testing.T) {
	service := NewAppService(nil)

	_, err := service.GetInteropStandards(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty scenario name")
	}
}

func TestGetInteropStandards_UnresolvableScenario(t *testing.T) {
	mockRepo := &mockAppRepository{apps: []repository.App{}}
	service := NewAppService(mockRepo)
	service.repoRoot = "/nonexistent/path"

	_, err := service.GetInteropStandards(context.Background(), "no-such-scenario")
	if err == nil {
		t.Fatal("expected error for unresolvable scenario")
	}
}

// =============================================================================
// 13. checkIframeGuard
// =============================================================================

func TestCheckIframeGuard_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
if (window.parent !== window) {
  initIframeBridgeChild({ appId: 'test' });
}
`)

	r := checkIframeGuard(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_iframe_guard" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckIframeGuard_Fail(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
initIframeBridgeChild({ appId: 'test' });
`)

	r := checkIframeGuard(tmp)
	if r.Passed {
		t.Error("expected fail when no iframe guard present")
	}
	if !strings.Contains(r.Message, "not guarded") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckIframeGuard_PassWindowTopSelf(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
if (window.top !== window.self) {
  initIframeBridgeChild({ appId: 'test' });
}
`)

	r := checkIframeGuard(tmp)
	if !r.Passed {
		t.Errorf("expected pass for window.top !== window.self guard, got fail: %s", r.Message)
	}
}

func TestCheckIframeGuard_Skip_NoBridgeCall(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import ReactDOM from 'react-dom/client';
ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
`)

	r := checkIframeGuard(tmp)
	if !r.Skipped {
		t.Error("expected skip when no bridge call found")
	}
}

// =============================================================================
// 14. checkCaptureEnabled
// =============================================================================

func TestCheckCaptureEnabled_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
initIframeBridgeChild({ appId: 'test' });
`)

	r := checkCaptureEnabled(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_capture_enabled" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckCaptureEnabled_FailCaptureLogs(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
initIframeBridgeChild({ appId: 'test', captureLogs: false });
`)

	r := checkCaptureEnabled(tmp)
	if r.Passed {
		t.Error("expected fail when captureLogs is disabled")
	}
	if !strings.Contains(r.Message, "captureLogs") {
		t.Errorf("expected message to mention captureLogs, got: %s", r.Message)
	}
}

func TestCheckCaptureEnabled_FailCaptureNetwork(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
initIframeBridgeChild({ appId: 'test', captureNetwork: { enabled: false } });
`)

	r := checkCaptureEnabled(tmp)
	if r.Passed {
		t.Error("expected fail when captureNetwork is disabled")
	}
	if !strings.Contains(r.Message, "captureNetwork") {
		t.Errorf("expected message to mention captureNetwork, got: %s", r.Message)
	}
}

func TestCheckCaptureEnabled_FailBoth(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
initIframeBridgeChild({ appId: 'test', captureLogs: false, captureNetwork: false });
`)

	r := checkCaptureEnabled(tmp)
	if r.Passed {
		t.Error("expected fail when both are disabled")
	}
	if !strings.Contains(r.Message, "captureLogs") || !strings.Contains(r.Message, "captureNetwork") {
		t.Errorf("expected message to mention both, got: %s", r.Message)
	}
}

func TestCheckCaptureEnabled_Skip_NoEntryFile(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/other.tsx", `export default function Foo() {}`)

	r := checkCaptureEnabled(tmp)
	if !r.Skipped {
		t.Error("expected skip when no entry file found")
	}
}

func TestCheckCaptureEnabled_Skip_NoBridgeCall(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/main.tsx", `
import ReactDOM from 'react-dom/client';
ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
`)

	r := checkCaptureEnabled(tmp)
	if !r.Skipped {
		t.Error("expected skip when no bridge call in entry file")
	}
	if r.SkipReason == "" || !strings.Contains(r.SkipReason, "initIframeBridgeChild") {
		t.Errorf("expected skip reason about missing bridge call, got: %s", r.SkipReason)
	}
}

// =============================================================================
// 15. checkProxyBasePreservation
// =============================================================================

func TestCheckProxyBasePreservation_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/api/client.ts", `
import { resolveApiBase } from '@vrooli/api-base';
const API_BASE = resolveApiBase();
fetch(API_BASE + '/health');
`)

	r := checkProxyBasePreservation(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_proxy_base_preserved" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckProxyBasePreservation_Fail(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/api/client.ts", `
import { resolveApiBase } from '@vrooli/api-base';
const API_BASE_INPUT = resolveApiBase();
const origin = window.location.origin;
const API_BASE_URL = origin + '/api/v1';
`)

	r := checkProxyBasePreservation(tmp)
	if r.Passed {
		t.Error("expected fail when window.location.origin is captured alongside resolveApiBase")
	}
	if !strings.Contains(r.Message, "window.location.origin") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckProxyBasePreservation_NoResolveApiBase(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/src/App.tsx", `
const origin = window.location.origin;
export default function App() { return null; }
`)

	r := checkProxyBasePreservation(tmp)
	if !r.Passed {
		t.Errorf("expected pass when no resolveApiBase present, got fail: %s", r.Message)
	}
}

func TestCheckProxyBasePreservation_SkipNoSrcDir(t *testing.T) {
	tmp := t.TempDir()

	r := checkProxyBasePreservation(tmp)
	if !r.Skipped {
		t.Error("expected skip when no ui/src/ directory")
	}
}

// =============================================================================
// 16. checkSecureTunnel
// =============================================================================

func TestCheckSecureTunnel_Pass(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/server.js", `
const express = require('express');
const app = express();
function proxyToApi(req, res, apiPath) {
  // proxy implementation
}
app.use('/api', (req, res) => proxyToApi(req, res, req.url));
app.listen(3000);
`)

	r := checkSecureTunnel(tmp)
	if !r.Passed {
		t.Errorf("expected pass, got fail: %s", r.Message)
	}
	if r.CheckID != "interop_secure_tunnel" {
		t.Errorf("unexpected check_id: %s", r.CheckID)
	}
}

func TestCheckSecureTunnel_Fail(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/server.js", `
const express = require('express');
const app = express();
app.use('/api', (req, res) => {
  fetch('http://localhost:3000' + req.url).then(r => r.json()).then(d => res.json(d));
});
app.listen(3000);
`)

	r := checkSecureTunnel(tmp)
	if r.Passed {
		t.Error("expected fail when custom server lacks proxyToApi")
	}
	if !strings.Contains(r.Message, "does not define proxyToApi") {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestCheckSecureTunnel_Skip_NoServerFile(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/package.json", `{}`)

	r := checkSecureTunnel(tmp)
	if !r.Skipped {
		t.Error("expected skip when no server file exists")
	}
}

func TestCheckSecureTunnel_Skip_StandardServer(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/server.js", `
import { startScenarioServer } from '@vrooli/api-base/server';
startScenarioServer();
`)

	r := checkSecureTunnel(tmp)
	if !r.Skipped {
		t.Error("expected skip when server is standard (no custom patterns)")
	}
}

// =============================================================================
// runAllInteropChecks - verify all 16 checks execute
// =============================================================================

func TestRunAllInteropChecks_ReturnsAll16(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp, "ui/package.json", `{"dependencies":{}}`)
	writeTestFile(t, tmp, "ui/vite.config.ts", `export default {}`)
	writeTestFile(t, tmp, "ui/src/main.tsx", `ReactDOM.render()`)

	results := runAllInteropChecks(tmp)
	if len(results) != 16 {
		t.Errorf("expected 16 check results, got %d", len(results))
	}

	// Verify unique check IDs
	seen := make(map[string]bool)
	for _, r := range results {
		if seen[r.CheckID] {
			t.Errorf("duplicate check_id: %s", r.CheckID)
		}
		seen[r.CheckID] = true
	}
}
