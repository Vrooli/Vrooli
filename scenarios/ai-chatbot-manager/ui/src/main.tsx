import React from 'react';
import ReactDOM from 'react-dom/client';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import App from './App';
import './index.css';

if (typeof window !== 'undefined' && window.parent !== window && !(window as any).__aiChatbotManagerBridgeInitialized) {
  let parentOrigin: string | undefined;

  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[AIChatbotManager] Unable to parse parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'ai-chatbot-manager' });
  (window as any).__aiChatbotManagerBridgeInitialized = true;
}

const rootElement = document.getElementById('root');

if (!rootElement) {
  throw new Error('Root element #root not found.');
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
