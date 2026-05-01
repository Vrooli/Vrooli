import assert from "node:assert/strict";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import type { useQuery } from "@tanstack/react-query";
import { ProfileActivityTable } from "../../src/features/stats/components/tables/ProfileActivityTable.js";
import type { ProfileBreakdownResponse } from "../../src/features/stats/api/types.js";
import { useProfileBreakdown } from "../../src/features/stats/hooks/useProfileBreakdown.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeProfileBreakdownResponse } from "../testutil/stats.js";

vi.mock("../../src/features/stats/hooks/useProfileBreakdown.js", () => ({
  useProfileBreakdown: vi.fn(),
}));

type QueryResult = ReturnType<typeof useQuery<ProfileBreakdownResponse, Error>>;

function queryResult(overrides: Partial<QueryResult>): QueryResult {
  return {
    data: undefined,
    isLoading: false,
    error: null,
    ...overrides,
  } as QueryResult;
}

function bodyProfileNames(): string[] {
  return screen
    .getAllByRole("row")
    .slice(1)
    .map((row) => within(row).getAllByRole("cell")[0]?.textContent ?? "");
}

test("ProfileActivityTable renders profile metrics and profile links", () => {
  vi.mocked(useProfileBreakdown).mockReturnValue(queryResult({
    data: makeProfileBreakdownResponse(),
  }));

  renderWithProviders(createElement(ProfileActivityTable));

  assert.ok(screen.getByText("Profile Activity"));
  assert.deepEqual(bodyProfileNames(), ["Maintenance Agent", "Implementation Agent", "Audit Agent"]);
  assert.ok(screen.getByText("85.7%"));
  assert.ok(screen.getByText("100.0%"));
  assert.ok(screen.getByText("33.3%"));
  assert.ok(screen.getByText("$7.50"));
  assert.equal(
    screen.getByRole("link", { name: "Maintenance Agent" }).getAttribute("href"),
    "/profiles?profileId=profile-maintenance",
  );
});

test("ProfileActivityTable sorts by profile name and cost from visible headers", async () => {
  const user = userEvent.setup();
  vi.mocked(useProfileBreakdown).mockReturnValue(queryResult({
    data: makeProfileBreakdownResponse(),
  }));

  renderWithProviders(createElement(ProfileActivityTable));

  await user.click(screen.getByRole("button", { name: /profile/i }));
  assert.deepEqual(bodyProfileNames(), ["Maintenance Agent", "Implementation Agent", "Audit Agent"]);

  await user.click(screen.getByRole("button", { name: /profile/i }));
  assert.deepEqual(bodyProfileNames(), ["Audit Agent", "Implementation Agent", "Maintenance Agent"]);

  await user.click(screen.getByRole("button", { name: /cost/i }));
  assert.deepEqual(bodyProfileNames(), ["Maintenance Agent", "Implementation Agent", "Audit Agent"]);
});

test("ProfileActivityTable renders empty, loading, and error states", () => {
  vi.mocked(useProfileBreakdown).mockReturnValue(queryResult({
    data: makeProfileBreakdownResponse({ profiles: [] }),
  }));

  const { unmount } = renderWithProviders(createElement(ProfileActivityTable));
  assert.ok(screen.getByText("No profile data available"));

  unmount();
  vi.mocked(useProfileBreakdown).mockReturnValue(queryResult({
    isLoading: true,
  }));

  const loading = renderWithProviders(createElement(ProfileActivityTable));
  assert.equal(document.querySelectorAll(".animate-pulse").length, 4);

  loading.unmount();
  vi.mocked(useProfileBreakdown).mockReturnValue(queryResult({
    error: new Error("profile stats unavailable"),
  }));

  renderWithProviders(createElement(ProfileActivityTable));
  assert.ok(screen.getByText("Failed to load: profile stats unavailable"));
});
