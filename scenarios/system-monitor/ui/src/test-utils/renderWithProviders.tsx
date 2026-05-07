/**
 * renderWithProviders — canonical render helper for component tests.
 *
 * Wraps the rendered tree in the same providers App.tsx mounts:
 *   - ThemeProvider
 *   - ToastProvider
 *   - ErrorBoundary
 *   - MemoryRouter (so route-aware components work without a real browser URL)
 */
import { render, type RenderOptions, type RenderResult } from '@testing-library/react';
import type { ReactElement, ReactNode } from 'react';
import { MemoryRouter } from 'react-router-dom';

import { ErrorBoundary } from '../shared/components/ErrorBoundary';
import { ToastProvider } from '../shared/components/ToastProvider';
import { ThemeProvider } from '../shared/theme/ThemeProvider';

export interface ProviderRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  /** Initial entries for the in-memory router. Defaults to ['/']. */
  initialEntries?: string[];
}

export function renderWithProviders(
  ui: ReactElement,
  options: ProviderRenderOptions = {},
): RenderResult {
  const { initialEntries = ['/'], ...rest } = options;

  const Wrapper = ({ children }: { children: ReactNode }) => (
    <ThemeProvider>
      <ToastProvider>
        <ErrorBoundary>
          <MemoryRouter initialEntries={initialEntries}>{children}</MemoryRouter>
        </ErrorBoundary>
      </ToastProvider>
    </ThemeProvider>
  );

  return render(ui, { wrapper: Wrapper, ...rest });
}
