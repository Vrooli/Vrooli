import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import { bootstrap } from './bootstrap';
import './styles.css';

declare global {
  interface Window {
    __vrooliAssistantBridgeInitialized?: boolean;
  }
}

if (typeof window !== 'undefined' && window.parent !== window && !window.__vrooliAssistantBridgeInitialized) {
  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[VrooliAssistant] Unable to determine parent origin', error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'vrooli-assistant' });
  window.__vrooliAssistantBridgeInitialized = true;
}

const rootElement = document.getElementById('root');
if (rootElement) {
  bootstrap(rootElement);
} else {
  console.error('[VrooliAssistant] Unable to locate root element for bootstrap');
}
