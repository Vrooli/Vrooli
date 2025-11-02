import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
import { mountApp } from './renderApp';
import { logger } from './utils/logger';

// Initialize iframe communication bridge when embedded within App Monitor
if (window.top !== window.self) {
  initIframeBridgeChild();
}

const container = document.getElementById('root');

if (!container) {
  throw new Error('Failed to locate root element for Browser Automation Studio UI');
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
