import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './styles/global.css';

declare global {
  interface Window {
    __API_URL__?: string;
  }
}

declare const __API_URL__: string | undefined;

if (typeof window !== 'undefined') {
  window.__API_URL__ = typeof __API_URL__ === 'string' ? __API_URL__ : window.__API_URL__;
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
