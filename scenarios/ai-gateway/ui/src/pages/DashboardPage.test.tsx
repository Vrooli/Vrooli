import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { listProviderRoles } from "../api/gateway";
import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";
import { DashboardPage } from "./DashboardPage";

vi.mock("../api/gateway", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/gateway")>();
  const { makeGatewayApiMocks } = await import("../test-utils/mocks/gateway");
  return { ...actual, ...makeGatewayApiMocks() };
});

describe("DashboardPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("[REQ:AIGW-UI-DASHBOARD] renders provider, evidence, and conformance summary panels", async () => {
    renderWithProviders(<DashboardPage />);

    expect(screen.getByTestId(selectors.dashboard.summary)).toBeInTheDocument();
    expect(await screen.findByTestId(selectors.dashboard.routeEvents)).toBeInTheDocument();
    expect(await screen.findByTestId(selectors.dashboard.conformanceDebt)).toBeInTheDocument();
  });

  it("renders backend errors without hiding the dashboard shell", async () => {
    vi.mocked(listProviderRoles).mockRejectedValueOnce(new Error("inventory unavailable"));

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByTestId(selectors.dashboard.error)).toHaveTextContent("inventory unavailable");
    expect(screen.getByTestId(selectors.dashboard.summary)).toBeInTheDocument();
  });
});
