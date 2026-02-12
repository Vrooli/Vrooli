export type RuleSeverity = 'critical' | 'high' | 'medium' | 'low';

export interface InteropRuleDef {
  id: string;
  name: string;
  severity: RuleSeverity;
  slot: string;
  file: string;
  recommendation: string;
  why: string;
  goodExample?: string;
  badExample?: string;
}

export interface InteropSlotGroup {
  slot: string;
  file: string;
  description: string;
  rules: InteropRuleDef[];
}

export const INTEROP_SLOT_GROUPS: InteropSlotGroup[] = [
  {
    slot: '[A]',
    file: 'ui/package.json',
    description: 'Package dependencies',
    rules: [
      {
        id: 'interop_api_base_dep',
        name: 'API base dependency',
        severity: 'critical',
        slot: '[A]',
        file: 'ui/package.json',
        recommendation: 'Add @vrooli/api-base to ui/package.json dependencies',
        why: 'Without @vrooli/api-base, the UI cannot resolve API endpoints across localhost, tunnel, and proxy contexts. API calls will break in non-localhost deployments.',
        goodExample: '"@vrooli/api-base": "workspace:*"',
        badExample: '// No @vrooli/api-base in dependencies',
      },
      {
        id: 'interop_iframe_bridge_dep',
        name: 'Iframe bridge dependency',
        severity: 'critical',
        slot: '[A]',
        file: 'ui/package.json',
        recommendation: 'Add @vrooli/iframe-bridge to ui/package.json dependencies',
        why: 'Without @vrooli/iframe-bridge, the UI cannot communicate with app-monitor when embedded in an iframe. Storage shimming, log capture, and keyboard relay all depend on this package.',
        goodExample: '"@vrooli/iframe-bridge": "workspace:*"',
        badExample: '// No @vrooli/iframe-bridge in dependencies',
      },
    ],
  },
  {
    slot: '[B]',
    file: 'ui/vite.config.ts',
    description: 'Build configuration',
    rules: [
      {
        id: 'interop_relative_base',
        name: 'Relative Vite base',
        severity: 'critical',
        slot: '[B]',
        file: 'ui/vite.config.ts',
        recommendation: 'Set base: \'./\' in ui/vite.config.ts',
        why: 'When served through a proxy at /apps/<name>/proxy/, absolute asset URLs resolve to the domain root, breaking all JS/CSS loading. Relative base makes assets resolve from the current directory.',
        goodExample: 'base: \'./' + '\'',
        badExample: 'base: \'/\'  // or no base config',
      },
      {
        id: 'interop_protective_comments',
        name: 'Protective comments',
        severity: 'low',
        slot: '[B],[D]',
        file: 'ui/vite.config.ts, ui/src/main.tsx',
        recommendation: 'Add INTEROP-CRITICAL comments to ui/vite.config.ts and ui/src/main.tsx',
        why: 'Protective comments prevent future developers from accidentally removing interop-critical code without understanding why it exists.',
        goodExample: '// INTEROP-CRITICAL: Relative base for proxy/tunnel contexts',
      },
    ],
  },
  {
    slot: '[C]',
    file: 'ui/server.js',
    description: 'Scenario server',
    rules: [
      {
        id: 'interop_no_custom_server',
        name: 'Standard scenario server',
        severity: 'medium',
        slot: '[C]',
        file: 'ui/server.js',
        recommendation: 'Use startScenarioServer() instead of custom Express/http server',
        why: 'startScenarioServer() from @vrooli/api-base handles proxy-aware static file serving, CORS, and health endpoints. Custom servers miss these and break in proxy/tunnel contexts.',
        goodExample: `import { startScenarioServer } from '@vrooli/api-base/server';

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'my-scenario',
});`,
        badExample: `import express from 'express';
const app = express();
app.use(express.static('dist'));
app.listen(3000);`,
      },
      {
        id: 'interop_secure_tunnel',
        name: 'Secure UI tunnel',
        severity: 'high',
        slot: '[C]',
        file: 'ui/server.js',
        recommendation: 'Route API calls through proxyToApi in custom server files',
        why: 'When the UI is exposed via a Cloudflare tunnel, API calls to localhost fail from the browser. A proxyToApi route on the UI server forwards API requests internally, keeping the API unexposed.',
      },
    ],
  },
  {
    slot: '[D]',
    file: 'ui/src/main.tsx',
    description: 'Iframe bridge initialization',
    rules: [
      {
        id: 'interop_bridge_init',
        name: 'Bridge initialization',
        severity: 'critical',
        slot: '[D]',
        file: 'ui/src/main.tsx',
        recommendation: 'Call initIframeBridgeChild() in ui/src/main.tsx',
        why: 'The iframe bridge must initialize before React mounts so that storage shimming is in place before any component accesses localStorage, and the message channel is ready for host commands.',
        goodExample: `import { initIframeBridgeChild } from '@vrooli/iframe-bridge';

if (window.parent !== window) {
  initIframeBridgeChild({ parentOrigin, appId: 'my-scenario' });
}

const root = ReactDOM.createRoot(document.getElementById('root')!);
root.render(<App />);`,
        badExample: `// initIframeBridgeChild() called inside a useEffect
// or after ReactDOM.createRoot()`,
      },
      {
        id: 'interop_iframe_guard',
        name: 'Iframe guard',
        severity: 'high',
        slot: '[D]',
        file: 'ui/src/main.tsx',
        recommendation: 'Guard initIframeBridgeChild with if (window.parent !== window)',
        why: 'Without the guard, bridge initialization runs even on localhost where there is no parent frame, which can cause errors or unexpected behavior.',
        goodExample: `if (window.parent !== window) {
  initIframeBridgeChild({ appId: 'my-scenario' });
}`,
        badExample: `// No guard -- runs unconditionally
initIframeBridgeChild({ appId: 'my-scenario' });`,
      },
      {
        id: 'interop_bridge_app_id',
        name: 'Bridge appId parameter',
        severity: 'medium',
        slot: '[D]',
        file: 'ui/src/main.tsx',
        recommendation: 'Pass appId to initIframeBridgeChild() call',
        why: 'The appId lets the host (app-monitor) identify which embedded scenario sent a message, enabling correct routing of bridge events when multiple iframes are active.',
        goodExample: `initIframeBridgeChild({ appId: 'my-scenario' })`,
        badExample: `initIframeBridgeChild()  // No appId`,
      },
      {
        id: 'interop_capture_enabled',
        name: 'Capture settings enabled',
        severity: 'medium',
        slot: '[D]',
        file: 'ui/src/main.tsx',
        recommendation: 'Do not disable captureLogs or captureNetwork in bridge init',
        why: 'Log and network capture allow app-monitor to display scenario console output and network activity. Disabling them removes observability.',
        badExample: `initIframeBridgeChild({
  captureLogs: false,      // Hides console output
  captureNetwork: false,   // Hides network activity
})`,
      },
    ],
  },
  {
    slot: '[E]',
    file: 'ui/src/App.tsx',
    description: 'Router basename',
    rules: [
      {
        id: 'interop_router_basename',
        name: 'Proxy-aware router',
        severity: 'high',
        slot: '[E]',
        file: 'ui/src/App.tsx',
        recommendation: 'Add basename prop to BrowserRouter (or use MemoryRouter)',
        why: 'When served through the proxy at /apps/<name>/proxy/, React Router needs the proxy path as basename so navigate("/page") resolves to /apps/<name>/proxy/page instead of /page.',
        goodExample: `import { getProxyInfo } from '@vrooli/api-base';

function getRouterBasename(): string {
  const proxyInfo = getProxyInfo();
  const proxyPath = proxyInfo?.primary?.path ?? proxyInfo?.basePath;
  return proxyPath ? proxyPath.replace(/\\/+$/, '') : '';
}

<BrowserRouter basename={getRouterBasename()}>`,
        badExample: `<BrowserRouter>  // No basename -- breaks under proxy`,
      },
    ],
  },
  {
    slot: '[F]',
    file: 'ui/src/lib/api-client.ts',
    description: 'API URL resolution',
    rules: [
      {
        id: 'interop_hardcoded_localhost',
        name: 'No hardcoded localhost',
        severity: 'high',
        slot: '[F]',
        file: 'ui/src/lib/api-client.ts',
        recommendation: 'Replace hardcoded localhost:PORT with resolveApiBase() from @vrooli/api-base',
        why: 'Hardcoded localhost URLs break when the UI is accessed via tunnel or proxy. resolveApiBase() auto-detects the correct API endpoint for all deployment contexts.',
        goodExample: `import { resolveApiBase } from '@vrooli/api-base';
const API_BASE = resolveApiBase({ appendSuffix: true });`,
        badExample: `const API_BASE = 'http://localhost:3000/api/v1';`,
      },
      {
        id: 'interop_resolve_api_base_single',
        name: 'Single API base resolution',
        severity: 'high',
        slot: '[F]',
        file: 'ui/src/lib/api-client.ts',
        recommendation: 'Import resolveApiBase in at most 2 production files',
        why: 'Centralizing API base resolution in one file prevents inconsistencies. If resolveApiBase is called in many files, some may resolve differently or miss updates.',
        goodExample: `// ui/src/lib/api-client.ts (single source)
const API_BASE = resolveApiBase({ appendSuffix: true });
export function buildUrl(path: string) { ... }`,
        badExample: `// Scattered across multiple files:
// ui/src/features/auth/api.ts
const base = resolveApiBase();
// ui/src/features/data/api.ts
const base = resolveApiBase();`,
      },
      {
        id: 'interop_proxy_base_preserved',
        name: 'Proxy base preservation',
        severity: 'high',
        slot: '[F]',
        file: 'ui/src/lib/api-client.ts',
        recommendation: 'Use resolveApiBase output directly; do not rebuild with window.location.origin',
        why: 'resolveApiBase returns a proxy-relative path in proxy context. Rebuilding with window.location.origin discards the proxy prefix, breaking API routing.',
        goodExample: `const API_BASE = resolveApiBase({ appendSuffix: true });
// Use API_BASE directly`,
        badExample: `const resolved = resolveApiBase();
const API_BASE = window.location.origin + resolved;`,
      },
    ],
  },
  {
    slot: '[G]',
    file: 'ui/src/hooks/useKeyboardShortcuts.ts',
    description: 'Keyboard shortcuts',
    rules: [
      {
        id: 'interop_shortcut_relay',
        name: 'Shortcut iframe relay',
        severity: 'medium',
        slot: '[G]',
        file: 'ui/src/hooks/useKeyboardShortcuts.ts',
        recommendation: 'Use emitShortcutIntent from @vrooli/iframe-bridge in keyboard hooks',
        why: 'When running in an iframe, unhandled keyboard shortcuts should be relayed to the host so app-monitor can handle them (e.g., Ctrl+K for global search).',
        goodExample: `import { emitShortcutIntent } from '@vrooli/iframe-bridge';

if (!handled) {
  emitShortcutIntent({
    action: HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
    outcome: 'noop',
    chord: 'mod+k',
  });
}`,
      },
      {
        id: 'interop_no_scattered_keydown',
        name: 'Centralized keyboard handling',
        severity: 'medium',
        slot: '[G]',
        file: 'ui/src/hooks/useKeyboardShortcuts.ts',
        recommendation: "Move app-level addEventListener('keydown') to hooks/ directory",
        why: 'Scattered keydown listeners across components make it impossible to ensure shortcuts are relayed to the host. A single central hook owns all app-level shortcuts.',
        goodExample: `// One hook in hooks/useKeyboardShortcuts.ts
export function useKeyboardShortcuts(handlers) {
  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);
}`,
        badExample: `// Multiple components adding their own listeners
// ComponentA.tsx
useEffect(() => { window.addEventListener('keydown', ...) })
// ComponentB.tsx
useEffect(() => { window.addEventListener('keydown', ...) })`,
      },
    ],
  },
];

export const TOTAL_RULE_COUNT = INTEROP_SLOT_GROUPS.reduce(
  (sum, group) => sum + group.rules.length,
  0,
);
