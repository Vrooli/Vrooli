import { Profiler, StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.tsx'
import { onProfilerRender } from './lib/profiler'

// ╔══════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: Iframe bridge initialization              ║
// ║                                                              ║
// ║  Must run BEFORE React mount so that:                        ║
// ║  1. Storage shimming is in place before any component        ║
// ║     accesses localStorage/sessionStorage                     ║
// ║  2. The bridge message channel is ready for host commands    ║
// ║                                                              ║
// ║  The window.parent check ensures this is a no-op when        ║
// ║  running outside an iframe (localhost, tunnel).              ║
// ╚══════════════════════════════════════════════════════════════╝

declare global {
  interface Window {
    __systemMonitorBridgeInitialized?: boolean;
  }
}

const initBridge = async () => {
  if (
    typeof window === 'undefined' ||
    window.parent === window ||
    window.__systemMonitorBridgeInitialized
  ) {
    return;
  }

  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch {
    // Fall back to default origin when parsing fails.
  }

  const { initIframeBridgeChild } = await import('@vrooli/iframe-bridge');
  initIframeBridgeChild({ parentOrigin, appId: 'system-monitor' });
  window.__systemMonitorBridgeInitialized = true;
};

const mountApp = () => {
  const rootEl = document.getElementById('root');
  if (!rootEl) throw new Error('Root element not found');
  createRoot(rootEl).render(
    <StrictMode>
      {/* Top-level Profiler boundary. Inert in regular prod (react-dom strips
          the profiling hook); emits user_timing entries via onProfilerRender
          when the perf-build channel is active. See lib/profiler.ts. Add inner
          <Profiler> boundaries around heavy subtrees as needed; do not remove
          this one. */}
      <Profiler id="App" onRender={onProfilerRender}>
        <App />
      </Profiler>
    </StrictMode>,
  )
};

void initBridge().finally(mountApp);

window.setTimeout(() => {
  void import('@vrooli/iframe-bridge/spatial')
    .then(({ initSpatialNav }) => { initSpatialNav(); });
}, 2200);
