import React from 'react';
import ReactDOM from 'react-dom/client';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import App from './App';
import './styles.css';

const initializeIframeBridge = (): void => {
  if (typeof window === 'undefined') {
    return;
  }

  const isIframe = window.parent && window.parent !== window;
  if (!isIframe || (window as any).__scenarioAuthenticatorBridgeInitialized) {
    return;
  }

  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[ScenarioAuthenticator] Unable to determine parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({ appId: 'scenario-authenticator', parentOrigin });
  (window as any).__scenarioAuthenticatorBridgeInitialized = true;
};

initializeIframeBridge();

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
