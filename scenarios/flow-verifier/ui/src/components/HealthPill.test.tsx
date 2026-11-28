import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { renderWithProviders } from "../test-utils";

vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth: vi.fn() };
});

import { HealthPill } from "./HealthPill";

function renderHealthPill() {
  return renderWithProviders(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <HealthPill />
    </QueryClientProvider>,
  );
}

describe("HealthPill", () => {
  beforeEach(async () => {
    const { fetchHealth } = await import("../api/health");
    vi.mocked(fetchHealth).mockReset();
  });
  afterEach(() => cleanup());

  it("renders an OK label when health resolves", async () => {
    const { fetchHealth } = await import("../api/health");
    const { makeHealthResponse } = await import("../test-utils");
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
    renderHealthPill();
    await waitFor(() =>
      expect(screen.getByTestId("health-pill")).toHaveTextContent(/ok/i),
    );
  });

  it("renders an error label when health fails", async () => {
    const { fetchHealth } = await import("../api/health");
    vi.mocked(fetchHealth).mockRejectedValue(new Error("boom"));
    renderHealthPill();
    await waitFor(() =>
      expect(screen.getByTestId("health-pill")).toHaveTextContent(/offline|error/i),
    );
  });
});
