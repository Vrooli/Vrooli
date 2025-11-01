import React from 'react';
import ReactDOM from 'react-dom/client';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

import App from './App';
import './index.css';

declare global {
  interface Window {
    __dataStructurerBridgeInitialized?: boolean;
    DATA_STRUCTURER_CONFIG?: {
      apiBase?: string;
    };
  }
}

const BRIDGE_STATE_KEY = '__dataStructurerBridgeInitialized' as const;

if (typeof window !== 'undefined' && window.parent !== window && !window[BRIDGE_STATE_KEY]) {
  let parentOrigin: string | undefined;

  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[data-structurer] Unable to infer parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({
    parentOrigin,
    appId: 'data-structurer',
    captureLogs: { enabled: true, streaming: true },
    captureNetwork: { enabled: true, streaming: true },
  });
  window[BRIDGE_STATE_KEY] = true;
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
