import React from 'react';
import ReactDOM from 'react-dom/client';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
import App from './App';
import './styles/global.css';

declare global {
  interface Window {
    __API_URL__?: string;
    __eloSwipeBridgeInitialized?: boolean;
  }
}

declare const __API_URL__: string | undefined;

if (typeof window !== 'undefined') {
  window.__API_URL__ = typeof __API_URL__ === 'string' ? __API_URL__ : window.__API_URL__;

  if (window.parent !== window && !window.__eloSwipeBridgeInitialized) {
    let parentOrigin: string | undefined;
    try {
      if (document.referrer) {
        parentOrigin = new URL(document.referrer).origin;
      }
    } catch (error) {
      console.warn('[EloSwipe] Unable to parse parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'elo-swipe' });
    window.__eloSwipeBridgeInitialized = true;
  }
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
