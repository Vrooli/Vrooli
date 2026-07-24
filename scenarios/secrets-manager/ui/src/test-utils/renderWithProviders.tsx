import { render, type RenderOptions } from "@testing-library/react";
import type { ReactElement } from "react";

// Shared render boundary for component tests. Add application providers here
// when a tested component requires them, rather than duplicating wrappers.
export function renderWithProviders(ui: ReactElement, options?: RenderOptions) {
  return render(ui, options);
}
