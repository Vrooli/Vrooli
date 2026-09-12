/**
 * HealthCard accessibility regression tests.
 *
 * The health feature owns its query-backed loading/success/error UI, so
 * a11y coverage lives here instead of in the app-composition smoke.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { makeHealthResponse } from "../../test-utils/factories";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { HealthCard } from "./HealthCard";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return {
    ...actual,
    fetchHealth: vi.fn().mockResolvedValue(makeHealthResponse()),
  };
});

describe("HealthCard accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the success state without axe violations", async () => {
    const { container } = renderWithProviders(<HealthCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.health.statusValue)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders the error state without axe violations", async () => {
    const { fetchHealth } = await import("../../api/health");
    vi.mocked(fetchHealth).mockRejectedValueOnce(new Error("health unavailable"));

    const { container } = renderWithProviders(<HealthCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.health.error)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });
});
