import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ProfilesPage } from "../../src/pages/ProfilesPage.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

function renderProfiles(overrides: Partial<Parameters<typeof ProfilesPage>[0]> = {}, initialEntries?: string[]) {
  const onRefresh = vi.fn();
  renderWithProviders(createElement(ProfilesPage, {
    profiles: [], loading: false, error: null, onRefresh,
    onCreateProfile: vi.fn(), onUpdateProfile: vi.fn(), onDeleteProfile: vi.fn(),
    ...overrides,
  }), initialEntries ? { initialEntries } : undefined);
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

test("ProfilesPage preserves an actionable failed delete, then clears the selected profile after a confirmed delete", async () => {
  const profile = { id: "delete-profile", name: "Delete target", roleRef: "code.default", maxTurns: 10 } as never;
  const remove = vi.fn()
    .mockRejectedValueOnce(new Error("profile is referenced by an active run"))
    .mockResolvedValueOnce(undefined);
  vi.stubGlobal("confirm", vi.fn(() => true));
  renderProfiles({ profiles: [profile], onDeleteProfile: remove }, ["/profiles?profileId=delete-profile"]);

  await waitFor(() => assert.ok(screen.getByRole("button", { name: "Delete" })));
  fireEvent.click(screen.getByRole("button", { name: "Delete" }));
  await waitFor(() => assert.equal(remove.mock.calls.length, 1));
  assert.ok(screen.getAllByText("Delete target").length > 0);

  fireEvent.click(screen.getByRole("button", { name: "Delete" }));
  await waitFor(() => assert.equal(remove.mock.calls.length, 2));
  assert.ok(screen.getAllByText("Delete target").length > 0);
});

test("ProfilesPage normalizes and submits a reusable profile configuration", async () => {
  const createProfile = vi.fn().mockResolvedValue({ id: "new-profile" });
  renderProfiles({ onCreateProfile: createProfile });
  fireEvent.click(screen.getByRole("button", { name: /create profile/i }));
  fireEvent.change(screen.getByLabelText("Name *"), { target: { value: "  Investigation Operator  " } });
  fireEvent.change(screen.getByLabelText("Profile Key"), { target: { value: "investigate" } });
  fireEvent.change(screen.getByLabelText("Description"), { target: { value: "Finds reliability regressions" } });
  fireEvent.change(screen.getByLabelText("Max Turns"), { target: { value: "35" } });
  fireEvent.change(screen.getByLabelText("Timeout (minutes)"), { target: { value: "50" } });
  fireEvent.change(screen.getByLabelText("Reasoning Effort"), { target: { value: "high" } });
  fireEvent.click(screen.getAllByRole("button", { name: "Create Profile" })[1]!);
  await waitFor(() => assert.deepEqual(createProfile.mock.calls[0]?.[0], {
    name: "  Investigation Operator  ", profileKey: "investigate", description: "Finds reliability regressions",
    roleRef: "code.default", maxTurns: 35, sandboxMode: "protected", networkAccess: "localhost",
    timeoutMinutes: 50, effort: "high", features: { enableBrowser: false }, extraFlags: {},
  }));
  assert.equal(screen.queryByRole("heading", { name: "Create New Profile" }), null);
});

test("ProfilesPage edits and deletes a selected profile through its detail controls", async () => {
  const profile = { id: "profile-1", name: "Operator", description: "original", roleRef: "code.default", profileKey: "operator", maxTurns: 10, createdAt: undefined, updatedAt: undefined } as never;
  const update = vi.fn().mockResolvedValue(profile); const remove = vi.fn().mockResolvedValue(undefined);
  vi.stubGlobal("confirm", vi.fn(() => true));
  renderProfiles({ profiles: [profile], onUpdateProfile: update, onDeleteProfile: remove });
  fireEvent.click(screen.getByRole("button", { name: "Edit" }));
  fireEvent.change(screen.getByLabelText("Name *"), { target: { value: "Updated operator" } });
  fireEvent.click(screen.getByRole("button", { name: "Update Profile" }));
  await waitFor(() => assert.equal(update.mock.calls[0]?.[0], "profile-1"));
  assert.equal(update.mock.calls[0]?.[1].name, "Updated operator");
  fireEvent.click(screen.getByRole("button", { name: "Delete" }));
  await waitFor(() => assert.deepEqual(remove.mock.calls, [["profile-1"]]));
});

test("ProfilesPage resolves a profile-key deep link and keeps an actionable create error visible", async () => {
  const profile = { id: "profile-keyed", name: "Deep-linked profile", profileKey: "investigator", roleRef: "code.default", maxTurns: 25 } as never;
  const createProfile = vi.fn(async () => { throw new Error("profile key already exists"); });
  renderProfiles({ profiles: [profile], onCreateProfile: createProfile }, ["/profiles?profileKey=investigator"]);
  await waitFor(() => assert.equal(screen.getAllByText("Deep-linked profile").length, 2));
  fireEvent.click(screen.getByRole("button", { name: "New" }));
  fireEvent.change(screen.getByLabelText("Name *"), { target: { value: "Duplicate" } });
  fireEvent.click(screen.getByRole("button", { name: "Create Profile" }));
  await waitFor(() => assert.ok(screen.getByText("profile key already exists")));
  assert.ok(screen.getByRole("heading", { name: "Create New Profile" }));
});

test("ProfilesPage lets an operator search, filter, sort, and reset the profile list", async () => {
  const profiles = [
    { id: "z", name: "Zebra reviewer", description: "review only", roleRef: "review.default", profileKey: "zebra" },
    { id: "a", name: "Alpha investigator", description: "analysis", roleRef: "code.default", profileKey: "alpha" },
  ] as never[];
  renderProfiles({
    profiles: profiles as never,
    rolePolicyCatalog: { roles: [
      { roleRef: "code.default", description: "Code" },
      { roleRef: "review.default", description: "Review" },
    ] } as never,
  });

  fireEvent.change(screen.getByLabelText("Search profiles..."), { target: { value: "review" } });
  assert.equal(screen.getAllByText("Zebra reviewer").length, 2);
  assert.equal(screen.queryAllByText("Alpha investigator").length, 0);
  fireEvent.click(screen.getByRole("button", { name: "Clear search" }));

  fireEvent.click(screen.getByRole("button", { name: "Filter and sort options" }));
  fireEvent.click(screen.getByRole("button", { name: "Filter by role" }));
  fireEvent.click(screen.getByRole("button", { name: "Code" }));
  assert.equal(screen.getAllByText("Alpha investigator").length, 2);
  assert.equal(screen.queryAllByText("Zebra reviewer").length, 0);
});

test("ProfilesPage preserves detailed policy settings when editing a linked profile", async () => {
  const profile = {
    id: "linked-profile", name: "Constrained investigator", profileKey: "constrained",
    description: "Uses an explicit policy envelope", roleRef: "investigate.default", maxTurns: 42,
    networkAccess: 3, allowedTools: ["read", "search"], deniedTools: ["shell"],
    timeout: { seconds: 2700n, nanos: 0 }, effort: "xhigh", features: { enableBrowser: true },
    extraFlags: { codex: { flags: ["--full-auto"] } }, sandboxConfig: { mode: 2 },
  } as never;
  const update = vi.fn().mockResolvedValue(profile);
  renderProfiles({ profiles: [profile], onUpdateProfile: update }, ["/profiles?profileId=linked-profile"]);

  await waitFor(() => assert.equal(screen.getAllByText("Constrained investigator").length, 2));
  fireEvent.click(screen.getByRole("button", { name: "Edit" }));
  assert.equal((screen.getByLabelText("Name *") as HTMLInputElement).value, "Constrained investigator");
  assert.equal((screen.getByLabelText("Max Turns") as HTMLInputElement).value, "42");
  assert.equal((screen.getByLabelText("Timeout (minutes)") as HTMLInputElement).value, "45");
  assert.equal((screen.getByLabelText("Reasoning Effort") as HTMLSelectElement).value, "xhigh");
  assert.equal((screen.getByLabelText("Request browser automation when the resolved runner supports it") as HTMLInputElement).checked, true);

  fireEvent.click(screen.getByRole("button", { name: "Update Profile" }));
  await waitFor(() => assert.deepEqual(update.mock.calls[0], ["linked-profile", {
    name: "Constrained investigator", profileKey: "constrained", description: "Uses an explicit policy envelope",
    roleRef: "investigate.default", maxTurns: 42, sandboxMode: "protected", networkAccess: "full",
    allowedTools: ["read", "search"], deniedTools: ["shell"], timeoutMinutes: 45, effort: "xhigh",
    features: { enableBrowser: true }, extraFlags: { codex: ["--full-auto"] },
  }]));
});

test("ProfilesPage applies form policy controls and can reset a role filter", async () => {
  const create = vi.fn().mockResolvedValue({ id: "saved" });
  const profiles = [
    { id: "code", name: "Code profile", roleRef: "code.default" },
    { id: "review", name: "Review profile", roleRef: "review.default" },
  ] as never[];
  renderProfiles({
    profiles: profiles as never,
    onCreateProfile: create,
    rolePolicyCatalog: { defaultRole: "review.default", roles: [
      { roleRef: "code.default", description: "Code" },
      { roleRef: "review.default", description: "Review" },
    ] } as never,
  });

  fireEvent.click(screen.getByRole("button", { name: "New" }));
  fireEvent.change(screen.getByLabelText("Name *"), { target: { value: "Browser reviewer" } });
  fireEvent.change(screen.getByLabelText("Sandbox Mode"), { target: { value: "off" } });
  fireEvent.change(screen.getByLabelText("Network Access"), { target: { value: "full" } });
  fireEvent.click(screen.getByLabelText("Request browser automation when the resolved runner supports it"));
  fireEvent.click(screen.getByRole("button", { name: "Create Profile" }));
  await waitFor(() => expect(create.mock.calls[0]?.[0]).toEqual(expect.objectContaining({
    name: "Browser reviewer", roleRef: "review.default", sandboxMode: "off", networkAccess: "full",
    features: { enableBrowser: true },
  })));

  fireEvent.click(screen.getByRole("button", { name: "Filter and sort options" }));
  fireEvent.click(screen.getByRole("button", { name: "Filter by role" }));
  fireEvent.click(screen.getByRole("button", { name: "Code" }));
  assert.equal(screen.queryAllByText("Review profile").length, 0);
  fireEvent.click(screen.getByRole("button", { name: "Reset filters" }));
  assert.ok(screen.getAllByText("Review profile").length > 0);
});
