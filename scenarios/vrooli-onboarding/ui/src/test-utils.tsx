import { renderWithProviders, type ProviderRenderOptions } from "@vrooli/api-base/testing";
export { act, cleanup, fireEvent, renderHook, screen, waitFor, within } from "@testing-library/react";
export { renderWithProviders } from "@vrooli/api-base/testing";
export type { ProviderRenderOptions } from "@vrooli/api-base/testing";
import { vi } from "vitest";

/** Compatibility name for older tests; all rendering uses the canonical provider tree. */
export function renderWithQueryClient(
  ui: Parameters<typeof renderWithProviders>[0],
  options?: ProviderRenderOptions,
) {
  return renderWithProviders(ui, options);
}

/** Mocks globalThis.fetch to resolve with the given JSON body. */
export function mockFetchSuccess(body: unknown) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(body),
  });
}

/** Mocks globalThis.fetch to resolve with a non-ok status (simulates API error). */
export function mockFetchError(status = 500) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: false,
    status,
  });
}

/** Mocks globalThis.fetch to never resolve (simulates loading state). */
export function mockFetchPending() {
  globalThis.fetch = vi.fn().mockImplementation(() => new Promise(() => {}));
}
