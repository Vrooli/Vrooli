import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { PermissionPolicyTab } from "../../src/components/dialogs/SettingsDialog/PermissionPolicyTab.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

test("PermissionPolicyTab exposes loading/error states and protects reconciliation behind operator authorization", async () => {
  const base = { validate: vi.fn(), reload: vi.fn(), plan: vi.fn(), doctor: vi.fn(), reconcile: vi.fn() };
  const loading = renderWithProviders(createElement(PermissionPolicyTab, { policy: { ...base, loading: true, error: null, data: undefined } as never }));
  assert.ok(screen.getByText("Loading permission policy…"));
  loading.unmount();
  renderWithProviders(createElement(PermissionPolicyTab, { policy: { ...base, loading: false, error: "catalog unavailable", data: undefined } as never }));
  assert.ok(screen.getByText("catalog unavailable"));
});

test("PermissionPolicyTab runs read and write operations with notices and clears authorization after reconcile", async () => {
  const policy = {
    loading: false, error: null,
    data: { status: { status: { ready: true, path: "policy.json", activeDigest: "digest" } } },
    validate: vi.fn().mockResolvedValue({ valid: true, candidateDigest: "candidate" }),
    reload: vi.fn().mockResolvedValue({ activated: true }),
    plan: vi.fn().mockResolvedValue({ plan: null }),
    doctor: vi.fn().mockResolvedValue({ healthy: true, summary: "doctor healthy", plan: null }),
    reconcile: vi.fn().mockResolvedValue({ result: { success: true } }),
  };
  renderWithProviders(createElement(PermissionPolicyTab, { policy: policy as never }));
  const reconcile = screen.getByRole("button", { name: "Reconcile declared permissions" });
  assert.equal((reconcile as HTMLButtonElement).disabled, true);
  fireEvent.click(screen.getByRole("button", { name: "Validate" }));
  await waitFor(() => assert.ok(screen.getByText("Catalog valid: candidate")));
  fireEvent.click(screen.getByRole("button", { name: "Reload" }));
  await waitFor(() => assert.ok(screen.getByText("Catalog activated")));
  fireEvent.click(screen.getByRole("button", { name: "Plan" }));
  await waitFor(() => assert.ok(screen.getByText("Projection plan refreshed")));
  fireEvent.click(screen.getByRole("button", { name: "Doctor" }));
  await waitFor(() => assert.equal(screen.getAllByText("doctor healthy").length, 2));
  fireEvent.click(screen.getByText(/I am explicitly authorized/));
  await waitFor(() => assert.equal((screen.getByRole("button", { name: "Reconcile declared permissions" }) as HTMLButtonElement).disabled, false));
  fireEvent.click(screen.getByRole("button", { name: "Reconcile declared permissions" }));
  await waitFor(() => assert.equal(policy.reconcile.mock.calls.length, 1));
  assert.equal((screen.getByRole("button", { name: "Reconcile declared permissions" }) as HTMLButtonElement).disabled, true);
});
