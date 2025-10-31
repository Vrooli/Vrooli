import React from 'react';
import ReactDOM from 'react-dom/client';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import App from './App';
import './styles/index.css';

declare global {
  interface Window {
    __saasLandingManagerBridgeInitialized?: boolean;
  }
}

if (typeof window !== 'undefined' && window.parent !== window && !window.__saasLandingManagerBridgeInitialized) {
  let parentOrigin: string | undefined;

  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[SaaS Landing Manager] Unable to determine parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'saas-landing-manager' });
  window.__saasLandingManagerBridgeInitialized = true;
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
