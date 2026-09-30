import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { MaintenanceTab } from "../../src/components/dialogs/SettingsDialog/MaintenanceTab.js";
import { RolePolicyTab } from "../../src/components/dialogs/SettingsDialog/RolePolicyTab.js";
import { PermissionPolicyTab } from "../../src/components/dialogs/SettingsDialog/PermissionPolicyTab.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { PurgeTarget } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";

test("MaintenanceTab sends the selected destructive target set only after its parent preview flow", () => {
  const onPattern = vi.fn(); const onPreview = vi.fn();
  const { rerender } = renderWithProviders(createElement(MaintenanceTab, {
    purgePattern: "^test-.*", onPurgePatternChange: onPattern, loading: false, error: "invalid pattern", onPurgePreview: onPreview,
  }));
  assert.ok(screen.getByText("invalid pattern"));
  fireEvent.change(screen.getByLabelText("Regex Pattern"), { target: { value: "^demo" } });
  assert.deepEqual(onPattern.mock.calls, [["^demo"]]);
  fireEvent.click(screen.getByRole("button", { name: "Delete Runs" }));
  assert.deepEqual(onPreview.mock.calls, [[[PurgeTarget.RUNS], "Delete Runs"]]);
  fireEvent.click(screen.getByRole("button", { name: "Delete All" }));
  assert.deepEqual(onPreview.mock.calls.at(-1), [[PurgeTarget.PROFILES, PurgeTarget.TASKS, PurgeTarget.RUNS], "Delete All"]);
  rerender(createElement(MaintenanceTab, { purgePattern: "x", onPurgePatternChange: onPattern, loading: true, error: null, onPurgePreview: onPreview }));
  assert.equal((screen.getByRole("button", { name: "Delete Profiles" }) as HTMLButtonElement).disabled, true);
});

test("RolePolicyTab communicates loading, failure, unavailable, and activated catalog state", () => {
  const { rerender } = renderWithProviders(createElement(RolePolicyTab, { data: null, loading: true, error: null }));
  assert.ok(screen.getByText("Loading role policy…"));
  rerender(createElement(RolePolicyTab, { data: null, loading: false, error: "catalog unavailable" }));
  assert.ok(screen.getByText("catalog unavailable"));
  rerender(createElement(RolePolicyTab, { data: {} as never, loading: false, error: null }));
  assert.ok(screen.getByText("Role policy status is unavailable."));
  rerender(createElement(RolePolicyTab, {
    loading: false, error: null,
    data: {
      status: { ready: true, activeDigest: "digest-1", path: "/repo/roles.yaml", requirement: { required: true, reason: "production" }, lastReloadAttempt: { diagnostic: { code: "OK", message: "reloaded" } } },
      catalog: { metadata: { catalogId: "core-roles" }, defaultRole: "code.default", roles: [{ roleRef: "code.default", intent: "implementation", description: "Write code", candidates: [{ runnerType: "codex", resourceRole: "primary" }] }] },
    } as never,
  }));
  assert.ok(screen.getByText("Ready")); assert.ok(screen.getByText("core-roles"));
  assert.equal(screen.getAllByText(/code.default/).length, 2); assert.ok(screen.getByText(/Required because:/));
});

test("PermissionPolicyTab exposes policy evidence and operator outcomes without assuming optional API fields", async () => {
  const policy = {
    data: {
      status: {
        status: {
          ready: false, path: "/repo/policy.json", activeDigest: "digest-2",
          requirement: { required: true, reason: "protected runner" },
          lastReloadAttempt: { diagnostic: { code: "STALE", message: "reload needed" } },
        },
        lastReconcile: { success: false, hardEnforcementSatisfied: false, missingHardEnforcementRuleIds: ["deny-shell"] },
      },
      catalog: {
        catalog: { metadata: { catalogId: "policy-core" }, targetScopes: ["agent"], rules: [{ id: "deny-shell", action: "deny", requiresHardEnforcement: true, matcher: { kind: "command", pattern: "shell" }, rationale: "safe", owner: "security", targetScope: "agent" }] },
      },
    },
    loading: false, error: null,
    validate: vi.fn(async () => ({ valid: false, diagnostic: { message: "invalid catalog" } })),
    reload: vi.fn(async () => ({ activated: false })),
    plan: vi.fn(async () => ({ plan: { hardEnforcementSatisfied: false, missingHardEnforcementRuleIds: ["deny-shell"], resources: [{ runnerType: "codex", scope: "agent", status: "planned", enforcement: { permissions: "read", caveats: ["review"] }, drift: true, error: "native mismatch", nativePaths: ["/tmp/policy"], changes: ["add deny"] }] } })),
    doctor: vi.fn(async () => ({ healthy: false, summary: "doctor found drift", plan: undefined })),
    reconcile: vi.fn(async () => ({ result: { success: false } })),
  };
  const { rerender } = renderWithProviders(createElement(PermissionPolicyTab, { policy: policy as never }));
  assert.ok(screen.getByText("Declared state not ready"));
  assert.ok(screen.getByText("Last reconcile incomplete"));
  assert.ok(screen.getByText(/reload needed/));
  assert.ok(screen.getByText(/Required hard enforcement is not satisfied/));
  assert.ok(screen.getByText(/policy-core/));
  fireEvent.click(screen.getByRole("button", { name: "Validate" }));
  await waitFor(() => assert.ok(screen.getByText("invalid catalog")));
  fireEvent.click(screen.getByRole("button", { name: "Reload" }));
  await waitFor(() => assert.ok(screen.getByText("Catalog was not activated")));
  fireEvent.click(screen.getByRole("button", { name: "Plan" }));
  await waitFor(() => assert.ok(screen.getByText("Hard enforcement gap")));
  assert.ok(screen.getByText("native mismatch"));
  assert.ok(screen.getByText("Native targets: /tmp/policy"));
  fireEvent.click(screen.getByRole("button", { name: "Doctor" }));
  await waitFor(() => assert.equal(screen.getAllByText("doctor found drift").length, 2));
  fireEvent.click(screen.getByRole("checkbox"));
  fireEvent.click(screen.getByRole("button", { name: "Reconcile declared permissions" }));
  await waitFor(() => assert.ok(screen.getByText("Reconcile completed with partial failure")));
  assert.equal((screen.getByRole("button", { name: "Reconcile declared permissions" }) as HTMLButtonElement).disabled, true);

  rerender(createElement(PermissionPolicyTab, { policy: { ...policy, data: null, loading: true } as never }));
  assert.ok(screen.getByText("Loading permission policy…"));
  rerender(createElement(PermissionPolicyTab, { policy: { ...policy, data: null, loading: false, error: "permission service unavailable" } as never }));
  assert.ok(screen.getByText("permission service unavailable"));
});
