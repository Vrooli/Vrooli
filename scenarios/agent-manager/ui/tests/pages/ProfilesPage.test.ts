import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ProfilesPage } from "../../src/pages/ProfilesPage.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

function renderProfiles(overrides: Partial<Parameters<typeof ProfilesPage>[0]> = {}) {
  const onRefresh = vi.fn();
  renderWithProviders(createElement(ProfilesPage, {
    profiles: [], loading: false, error: null, onRefresh,
    onCreateProfile: vi.fn(), onUpdateProfile: vi.fn(), onDeleteProfile: vi.fn(),
    ...overrides,
  }));
  return onRefresh;
}

test("ProfilesPage exposes empty, error, refresh, and create-profile entry states", () => {
  const refresh = renderProfiles({ error: "Profiles API unavailable" });
  assert.ok(screen.getByText(/Agent Profiles \(0\)/));
  assert.ok(screen.getByText("No Agent Profiles"));
  assert.ok(screen.getByText("Profiles API unavailable"));
  fireEvent.click(screen.getAllByRole("button")[1]!);
  assert.equal(refresh.mock.calls.length, 1);
  fireEvent.click(screen.getByRole("button", { name: /create profile/i }));
  assert.ok(screen.getByRole("dialog"));
  assert.ok(screen.getByRole("heading", { name: "Create New Profile" }));
});
