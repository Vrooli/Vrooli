import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { fetchHealth } from "../api/health";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "../app/routes";

vi.mock("../api/health", () => ({
  fetchHealth: vi.fn(),
}));

const mockedFetchHealth = vi.mocked(fetchHealth);

describe("AppShell health utility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows healthy when the health endpoint responds", async () => {
    mockedFetchHealth.mockResolvedValueOnce({ status: "healthy" } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByTestId(selectors.health.statusBadge)).toHaveTextContent("Healthy"));
  });

  it("shows checking while the health endpoint is pending", () => {
    mockedFetchHealth.mockReturnValueOnce(new Promise(() => {}));
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.health.statusBadge)).toHaveTextContent("Checking");
  });

  it("shows unreachable when the health endpoint fails", async () => {
    mockedFetchHealth.mockRejectedValueOnce(new Error("offline"));
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByTestId(selectors.health.statusBadge)).toHaveTextContent("Unreachable"), { timeout: 5000 });
  });
});
