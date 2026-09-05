/**
 * HealthCard tests — focused on the health-card surface only.
 *
 * Renders <HealthCard /> directly (not through <App />) so failures point at
 * health-feature behaviour, not shell composition. The /health fetch is mocked
 * so the card reaches its success state deterministically.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { makeApiMocks, renderWithProviders } from "../../test-utils";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});

import { HealthCard } from "./HealthCard";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

describe("HealthCard rendering (cimode — copy-independent)", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the health-card root via its test id", () => {
    renderWithProviders(<HealthCard />);
    expect(screen.getByTestId(selectors.health.card)).toBeInTheDocument();
  });

  it("renders the health title via the strings registry", () => {
    renderWithProviders(<HealthCard />);
    expect(screen.getByText(strings.health.title)).toBeInTheDocument();
  });

  it("exposes the refresh button regardless of label copy", () => {
    renderWithProviders(<HealthCard />);
    expect(screen.getByTestId(selectors.health.refreshButton)).toBeInTheDocument();
  });

  it("surfaces the status/service/timestamp once /health resolves", async () => {
    renderWithProviders(<HealthCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.health.statusValue)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.health.serviceValue)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.health.timestampValue)).toBeInTheDocument();
  });
});
