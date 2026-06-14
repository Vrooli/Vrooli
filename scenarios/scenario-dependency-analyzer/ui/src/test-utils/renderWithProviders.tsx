import { type RenderOptions, type RenderResult, render } from "@testing-library/react";
import type { ReactElement } from "react";

import { Providers } from "../app/providers";

export function renderWithProviders(
  ui: ReactElement,
  options?: Omit<RenderOptions, "wrapper">
): RenderResult {
  return render(ui, {
    wrapper: ({ children }) => <Providers>{children}</Providers>,
    ...options
  });
}
