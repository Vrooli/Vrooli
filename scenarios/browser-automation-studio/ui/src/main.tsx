import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
import { mountApp } from './renderApp';

// Initialize iframe communication bridge when embedded within App Monitor
if (window.top !== window.self) {
  initIframeBridgeChild();
}

const container = document.getElementById('root');

if (!container) {
  throw new Error('Failed to locate root element for Browser Automation Studio UI');
}

mountApp(container, { strictMode: true });
