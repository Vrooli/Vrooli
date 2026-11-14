import type { ReactElement } from "react";
import { render as testingLibraryRender, type RenderOptions, type RenderResult } from "@testing-library/react";

/**
 * Shared render seam for Agent Inbox component tests. The current application
 * has no global provider stack; new providers belong here when introduced.
 */
export function renderWithProviders(ui: ReactElement, options?: RenderOptions): RenderResult {
  return testingLibraryRender(ui, options);
}

export { testingLibraryRender as render };
export type { RenderOptions, RenderResult };
