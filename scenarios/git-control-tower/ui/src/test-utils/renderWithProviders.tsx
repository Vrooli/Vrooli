import type { ReactElement } from "react";
import type { RenderOptions } from "@testing-library/react";
import { renderWithQueryClient, createTestQueryClient } from "./render";

// Canonical projection used by the scenario test policy. Keep the query
// provider wrapper in one place so feature tests do not invent variants.
export function renderWithProviders(ui: ReactElement, options: RenderOptions & { queryClient?: ReturnType<typeof createTestQueryClient> } = {}) {
  return renderWithQueryClient(ui, options);
}
