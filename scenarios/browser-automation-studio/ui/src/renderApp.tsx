import * as React from 'react';
import type { ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
// import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { Toaster } from 'react-hot-toast';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import { AppWithUpdates, ReadyMarker } from './AppShell';
import { WebSocketProvider } from './contexts/WebSocketProvider';
import './index.css';
import { onProfilerRender } from './lib/profiler';
import { logger } from './utils/logger';

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

function renderTree(): ReactNode {
  const content = (
    <QueryClientProvider client={queryClient}>
      <WebSocketProvider>
        <ReadyMarker />
        <React.Profiler id="App" onRender={onProfilerRender}>
          <AppWithUpdates />
        </React.Profiler>
        <Toaster
          position="bottom-right"
          toastOptions={{
            className: '',
            duration: 4000,
            style: {
              background: 'rgb(var(--flow-node))',
              color: 'rgb(var(--flow-text))',
              border: '1px solid rgb(var(--flow-border))',
            },
          }}
        />
        {/* <ReactQueryDevtools initialIsOpen={false} /> */}
      </WebSocketProvider>
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
