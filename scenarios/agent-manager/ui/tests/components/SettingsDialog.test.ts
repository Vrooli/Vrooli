import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { SettingsDialog } from "../../src/components/dialogs/SettingsDialog/index.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

vi.mock("../../src/hooks/useApi.js", () => ({
  useRolePolicyCatalog: () => ({ data: null, loading: false, error: null }),
  usePermissionPolicy: () => ({ data: null, loading: false, error: null }),
  useMaintenance: () => ({ previewPurge: vi.fn(), executePurge: vi.fn() }),
  useInvestigationSettings: () => ({ data: null, loading: false, error: null, updateSettings: vi.fn(), resetSettings: vi.fn() }),
}));
vi.mock("../../src/hooks/useOrchestrationSettings.js", () => ({ useOrchestrationSettings: () => ({ data: null, loading: false, error: null, updateSettings: vi.fn(), resetSettings: vi.fn() }) }));
vi.mock("../../src/components/dialogs/SettingsDialog/RolePolicyTab.js", () => ({ RolePolicyTab: () => "role policy" }));
vi.mock("../../src/components/dialogs/SettingsDialog/PermissionPolicyTab.js", () => ({ PermissionPolicyTab: () => "permission policy" }));
vi.mock("../../src/components/dialogs/SettingsDialog/ModelPricingTab.js", () => ({ ModelPricingTab: () => "pricing" }));
vi.mock("../../src/components/dialogs/SettingsDialog/MaintenanceTab.js", () => ({ MaintenanceTab: () => "maintenance" }));
vi.mock("../../src/components/dialogs/PurgeConfirmDialog.js", () => ({ PurgeConfirmDialog: () => null }));

test("SettingsDialog exposes every settings category with an accurate operator description and close control", async () => {
  const user = userEvent.setup();
  const onOpenChange = vi.fn();
  renderWithProviders(createElement(SettingsDialog, { open: true, onOpenChange, onPurgeComplete: vi.fn() }));
  assert.ok(screen.getByText("Inspect the active Git-managed role policy catalog and activation state"));
  for (const [tab, description] of [
    ["Permissions", "Inspect and reconcile global portable coding-agent permissions"],
    ["Model Pricing", "View and manage model pricing with overrides"],
    ["Investigation", "Configure investigation and apply-fix agent behavior"],
    ["Orchestration", "Configure run lifecycle, safety, health detection, and termination behavior"],
    ["Maintenance", "Purge data and manage service controls"],
  ]) {
    await user.click(screen.getByRole("tab", { name: tab }));
    assert.ok(screen.getByText(description));
  }
  await user.click(screen.getAllByRole("button", { name: "Close" })[1]!);
  assert.deepEqual(onOpenChange.mock.calls, [[false]]);
});
