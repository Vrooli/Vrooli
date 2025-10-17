import React from 'react';
import ReactDOM from 'react-dom/client';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import './index.css';
import App from './App';

const BRIDGE_STATE_KEY = '__resourceExperimenterBridgeInitialized';

function initializeIframeBridge() {
  if (!(typeof window !== 'undefined' && window.parent !== window)) {
    return;
  }

  if (window[BRIDGE_STATE_KEY]) {
    return;
  }

  let parentOrigin;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[ResourceExperimenter] Unable to parse parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'resource-experimenter' });
  window[BRIDGE_STATE_KEY] = true;
}

if (typeof document !== 'undefined') {
  if (document.readyState === 'complete' || document.readyState === 'interactive') {
    initializeIframeBridge();
  } else {
    document.addEventListener('DOMContentLoaded', initializeIframeBridge, { once: true });
  }
}

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
