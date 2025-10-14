import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './index.css';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const parentOrigin = document.referrer ? new URL(document.referrer).origin : undefined;

if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({ appId: 'kids-dashboard', parentOrigin });
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);