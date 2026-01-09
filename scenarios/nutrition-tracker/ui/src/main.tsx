import React from 'react';
import ReactDOM from 'react-dom/client';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

import App from './App';
import './index.css';

declare global {
  interface Window {
    __nutritionTrackerBridgeInitialized?: boolean;
  }
}

const BRIDGE_STATE_KEY = '__nutritionTrackerBridgeInitialized' as const;

if (typeof window !== 'undefined' && window.parent !== window && !window[BRIDGE_STATE_KEY]) {
  let parentOrigin: string | undefined;

  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[NutritionTracker] Unable to determine parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'nutrition-tracker' });
  window[BRIDGE_STATE_KEY] = true;
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
