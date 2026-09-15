import type { ReactElement, ReactNode } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { render as rtlRender, type RenderOptions, type RenderResult } from '@testing-library/react';

/**
 * Canonical component-test render seam. App Monitor currently owns routing as
 * its only cross-cutting provider; keeping it here makes tests explicit about
 * whether they need a fresh memory history and gives future providers one
 * projection point. Tests that already construct a deliberate nested router
 * may use the lower-level Testing Library render with a scoped rationale.
 */
export function renderWithProviders(
  ui: ReactElement,
  options: RenderOptions & { initialEntries?: string[] } = {},
): RenderResult {
  const { initialEntries = ['/'], ...renderOptions } = options;
  const tree: ReactNode = <MemoryRouter initialEntries={initialEntries}>{ui}</MemoryRouter>;
  return rtlRender(tree, renderOptions);
}

export { rtlRender as render };
export type { RenderResult, RenderOptions };
