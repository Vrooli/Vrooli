import React from 'react';
import ReactDOM from 'react-dom/client';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import App from './App';
import './index.css';

declare global {
  interface Window {
    __coreDebuggerBridgeInitialized?: boolean;
  }
}

if (typeof window !== 'undefined' && window.parent !== window && !window.__coreDebuggerBridgeInitialized) {
  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[CoreDebugger] Unable to parse parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'core-debugger' });
  window.__coreDebuggerBridgeInitialized = true;
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
