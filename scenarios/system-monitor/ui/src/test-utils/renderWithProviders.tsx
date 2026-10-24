import { render, type RenderOptions, type RenderResult } from '@testing-library/react';
import { getProxyInfo } from '@vrooli/api-base';
import type { ReactElement } from 'react';
import { BrowserRouter } from 'react-router-dom';
import { TimeRangeProvider } from '../shared/time/TimeRangeContext';
import { ThemeProvider } from '../shared/theme/ThemeProvider';
import { ToastProvider } from '../shared/components/ToastProvider';

function TestProviders({ children }: { children: React.ReactNode }) {
  const proxyInfo = getProxyInfo();
  const proxyPath = proxyInfo?.primary?.path ?? proxyInfo?.basePath;
  const basename = proxyPath ? proxyPath.replace(/\/+$/, '') : '';

  return (
    <ThemeProvider>
      <ToastProvider>
        <TimeRangeProvider>
          <BrowserRouter basename={basename}>{children}</BrowserRouter>
        </TimeRangeProvider>
      </ToastProvider>
    </ThemeProvider>
  );
}

export function renderWithProviders(ui: ReactElement, options?: Omit<RenderOptions, 'wrapper'>): RenderResult {
  return render(ui, { ...options, wrapper: TestProviders });
}
