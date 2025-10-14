import React from 'react';
import ReactDOM from 'react-dom/client';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import './styles/index.css';
import App from './App';
import reportWebVitals from './reportWebVitals';

const BRIDGE_STATE_KEY = '__reactComponentLibraryBridgeInitialized';

if (typeof window !== 'undefined' && window.parent !== window) {
  const globalWindow = window as typeof window & {
    __reactComponentLibraryBridgeInitialized?: boolean;
  };

  if (!globalWindow.__reactComponentLibraryBridgeInitialized) {
    let parentOrigin: string | undefined;
    try {
      if (document.referrer) {
        parentOrigin = new URL(document.referrer).origin;
      }
    } catch (error) {
      console.warn('[ReactComponentLibrary] Unable to determine parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'react-component-library' });
    globalWindow.__reactComponentLibraryBridgeInitialized = true;
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
