import React from 'react';
import ReactDOM from 'react-dom/client';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import App from './App';

if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({ appId: 'scenario-to-mcp-ui' });
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
