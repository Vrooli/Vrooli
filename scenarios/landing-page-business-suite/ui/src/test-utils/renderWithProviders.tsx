import type { ReactElement, ReactNode } from "react";
import { render, type RenderOptions } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

type Options = Omit<RenderOptions, "wrapper"> & { route?: string };

// renderWithProviders is the single canonical UI render seam. Add global test
// providers here as the application adopts them instead of rebuilding wrappers
// in individual tests.
export function renderWithProviders(ui: ReactElement, options: Options = {}) {
  const { route, ...renderOptions } = options;
  function Wrapper({ children }: { children: ReactNode }) {
    // Existing component tests frequently supply BrowserRouter/MemoryRouter as
    // part of the subject under test. Route injection is therefore opt-in; the
    // helper remains the single extension point for common providers without
    // introducing a second router around those subjects.
    return route === undefined ? <>{children}</> : <MemoryRouter initialEntries={[route]}>{children}</MemoryRouter>;
  }
  return render(ui, { wrapper: Wrapper, ...renderOptions });
}
