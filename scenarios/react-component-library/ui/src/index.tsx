import React from 'react';
import ReactDOM from 'react-dom/client';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
import './styles/index.css';
import App from './App';
import reportWebVitals from './reportWebVitals';

declare global {
  interface Window {
    __reactComponentLibraryBridgeInitialized?: boolean;
  }
}

const shouldInitIframeBridge = typeof window !== 'undefined' && (() => {
  try {
    return window.self !== window.top;
  } catch (error) {
    // Accessing window.top can throw in cross-origin iframe scenarios; treat as inside iframe
    console.debug('[ReactComponentLibrary] Unable to read window.top when checking iframe status', error);
    return true;
  }
})();

if (shouldInitIframeBridge && !window.__reactComponentLibraryBridgeInitialized) {
  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[ReactComponentLibrary] Unable to determine parent origin for iframe bridge', error);
  }

  try {
    initIframeBridgeChild({
      parentOrigin,
      appId: 'react-component-library',
      captureLogs: { enabled: true, streaming: true },
      captureNetwork: { enabled: true, streaming: true },
    });
    window.__reactComponentLibraryBridgeInitialized = true;
  } catch (error) {
    console.error('[ReactComponentLibrary] Failed to initialize iframe bridge', error);
  }
}

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
