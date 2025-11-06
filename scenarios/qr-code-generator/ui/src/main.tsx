import React from 'react';
import ReactDOM from 'react-dom/client';

import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import { cacheBridgeController } from '@/lib/iframeBridge';
import App from './App';
import './index.css';

const BRIDGE_APP_ID = 'qr-code-generator';
const LOG_LEVELS: Array<'log' | 'info' | 'warn' | 'error' | 'debug'> = [
  'log',
  'info',
  'warn',
  'error',
  'debug'
];

function resolveParentOrigin(): string | undefined {
  try {
    if (document.referrer) {
      return new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[QR UI] Unable to determine parent origin for iframe bridge', error);
  }

  return undefined;
}

function bootstrapIframeBridge() {
  if (typeof window === 'undefined') {
    return;
  }

  if (window.parent === window) {
    cacheBridgeController(null);
    return;
  }

  if (window.__qrCodeGeneratorBridgeInitialized && window.__qrCodeGeneratorBridgeController) {
    cacheBridgeController(window.__qrCodeGeneratorBridgeController);
    return;
  }

  try {
    const controller = initIframeBridgeChild({
      parentOrigin: resolveParentOrigin(),
      appId: BRIDGE_APP_ID,
      captureLogs: {
        enabled: true,
        streaming: true,
        bufferSize: 400,
        levels: LOG_LEVELS
      },
      captureNetwork: {
        enabled: true,
        streaming: true,
        bufferSize: 200
      }
    });
    cacheBridgeController(controller);
  } catch (error) {
    console.error('[QR UI] Failed to bootstrap iframe bridge', error);
    cacheBridgeController(null);
  }
}

bootstrapIframeBridge();

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
