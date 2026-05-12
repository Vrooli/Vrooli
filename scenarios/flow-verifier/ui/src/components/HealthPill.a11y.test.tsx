import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { expect } from "vitest";

import { expectNoA11yViolations, renderWithProviders, makeHealthResponse } from "../test-utils";

vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth: vi.fn() };
});

import { HealthPill } from "./HealthPill";

describe("HealthPill accessibility", () => {
  beforeEach(async () => {
    const { fetchHealth } = await import("../api/health");
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
  });
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(<HealthPill />);
    await waitFor(() =>
      expect(screen.getByTestId("health-pill")).toHaveTextContent(/ok/i),
    );
    await expectNoA11yViolations(container);
  });
});
