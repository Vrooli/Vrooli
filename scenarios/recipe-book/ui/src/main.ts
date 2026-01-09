import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import { bootstrap } from './bootstrap';
import './styles.css';

declare global {
  interface Window {
    __recipeBookBridgeInitialized?: boolean;
  }
}

function initializeBridge(): void {
  if (typeof window === 'undefined' || window.parent === window || window.__recipeBookBridgeInitialized) {
    return;
  }

  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[RecipeBook] Unable to parse parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'recipe-book' });
  window.__recipeBookBridgeInitialized = true;
}

initializeBridge();

const rootElement = document.getElementById('root');
if (rootElement) {
  bootstrap(rootElement);
} else {
  console.error('[RecipeBook] Failed to locate root element for bootstrap');
}
