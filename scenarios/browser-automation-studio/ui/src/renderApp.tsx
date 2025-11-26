import * as React from 'react';
import type { ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
// import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { Toaster } from 'react-hot-toast';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import App from './App';
import './index.css';
import { logger } from './utils/logger';
import { ensureReadyMarker, markAppReady } from './ready';

declare global {
  interface Window {
    __browserAutomationStudioBridgeInitialized?: boolean;
  }
}

interface MountOptions {
  strictMode?: boolean;
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 5 * 60 * 1000, // 5 minutes
    },
  },
});

if (typeof window !== 'undefined') {
  // Expose React for libraries that expect it on the global scope in embedded contexts
  (window as typeof window & { React?: typeof React }).React = React;
}

let bridgeInitialized = false;

function ensureBridge() {
  if (typeof window === 'undefined') {
    return;
  }

  if (window.parent === window) {
    return;
  }

  if (bridgeInitialized || window.__browserAutomationStudioBridgeInitialized) {
    return;
  }

  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    logger.warn('Unable to parse parent origin for iframe bridge', { component: 'renderApp', action: 'ensureBridge' }, error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'browser-automation-studio' });
  window.__browserAutomationStudioBridgeInitialized = true;
  bridgeInitialized = true;
}

function ReadyMarker(): null {
  React.useEffect(() => {
    ensureReadyMarker();
    // Slightly defer setting the ready flag until after first paint.
    requestAnimationFrame(() => markAppReady());
  }, []);
  return null;
}

function renderTree(): ReactNode {
  const content = (
    <QueryClientProvider client={queryClient}>
      <ReadyMarker />
      <App />
      <Toaster
        position="bottom-right"
        toastOptions={{
          className: '',
          duration: 4000,
          style: {
            background: '#1a1d29',
            color: '#fff',
            border: '1px solid #2a2d3a',
          },
        }}
      />
      {/* <ReactQueryDevtools initialIsOpen={false} /> */}
    </QueryClientProvider>
  );

  return content;
}

export function mountApp(target: HTMLElement, options: MountOptions = {}): Root {
  ensureBridge();

  const { strictMode = false } = options;
  const tree = strictMode ? <React.StrictMode>{renderTree()}</React.StrictMode> : renderTree();

  const root = createRoot(target);
  root.render(tree);
  return root;
}

export default mountApp;
