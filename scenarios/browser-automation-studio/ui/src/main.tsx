import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
import { mountApp } from './renderApp';
import { logger } from './utils/logger';

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
    __browserAutomationStudioBridgeInitialized?: boolean;
  }
}

if (
  typeof window !== 'undefined' &&
  window.parent !== window &&
  !window.__browserAutomationStudioBridgeInitialized
) {
  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch {
    // Fall back to default origin when parsing fails.
  }

  initIframeBridgeChild({ parentOrigin, appId: 'browser-automation-studio' });
  window.__browserAutomationStudioBridgeInitialized = true;
}

const container = document.getElementById('root');

if (!container) {
  throw new Error('Failed to locate root element for Vrooli Ascension UI');
}

const pathname = window.location.pathname || '';

if (pathname.startsWith('/export/replay') || pathname.startsWith('/export/composer')) {
  void import('./export/exportBootstrap')
    .then(({ mountReplayExport }) => {
      mountReplayExport(container);
    })
    .catch((error) => {
      logger.error('Failed to bootstrap replay export view', { component: 'main' }, error);
    });
} else {
  mountApp(container, { strictMode: true });
}
