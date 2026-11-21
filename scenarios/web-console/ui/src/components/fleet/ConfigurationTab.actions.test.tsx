import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApplyRunState, ApplyStepState } from "@vrooli/proto-types/vrooli-onboarding/v1/apply/apply_pb";
import { renderWithProviders as render } from "../../test-utils";
import type { Machine } from "../../api/machines";

const apiMocks = vi.hoisted(() => ({
  getConfiguration: vi.fn(),
  listCredentialGrants: vi.fn(),
  reapplyConfiguration: vi.fn(),
  getConfigurationApplyStatus: vi.fn(),
}));

vi.mock("../../api/machines", async () => {
  const actual = await vi.importActual<typeof import("../../api/machines")>("../../api/machines");
  return { ...actual, ...apiMocks };
});

import { ConfigurationTab } from "./ConfigurationTab";

/** minimouse as it looked when every Fix button ran the same re-apply. */
function machine(): Machine {
  return {
    target: {
      id: "bridge-node:minimouse",
      kind: "bridge-node",
      label: "minimouse",
      available: true,
      readiness: [
        { key: "capability:agy", label: "Antigravity", passed: true, state: "ready" },
        { key: "capability:opencode", label: "OpenCode", passed: false, state: "missing", detail: "command is not on PATH" },
        {
          key: "node_capability:bridge-provisioner",
          label: "Remote updates (Bridge provisioning)",
          passed: false,
          state: "missing",
          detail: "helper_not_installed: the privileged provisioning helper is not installed",
        },
      ],
    },
    grant: { summary: "Read terminal", effects: ["read"], appCount: 1, coversAllApps: false, scopes: [], preset: "read" },
    heartbeatAgeSeconds: 4,
    manageable: true,
    drift: [{ kind: "profile", name: "managed-connection", reason: "profile has not been applied" }],
  } as Machine;
}

beforeEach(() => {
  vi.clearAllMocks();
  apiMocks.getConfiguration.mockResolvedValue({ questions: [], readiness: null, detail: null });
  apiMocks.listCredentialGrants.mockResolvedValue({ grants: [] });
});

describe("ConfigurationTab drift actions", () => {
  it("installs a missing agent from its own row and shows the machine's reason when it fails", async () => {
    const onInstall = vi.fn().mockResolvedValue({ status: "failed", message: "Bridge could not run the opencode installer on minimouse" });
    render(<ConfigurationTab machine={machine()} onInstallCapability={onInstall} />);

    fireEvent.click(await screen.findByTestId("machine-drift-install-opencode"));

    await waitFor(() => {
      expect(screen.getByTestId("machine-drift-capability-opencode-status")).toHaveTextContent("Bridge could not run the opencode installer");
    });
    expect(onInstall).toHaveBeenCalledWith("opencode", expect.objectContaining({ id: "bridge-node:minimouse" }));
    expect(apiMocks.reapplyConfiguration).not.toHaveBeenCalled();
  });

  it("offers no action for a node feature a browser cannot install, and explains why it is missing", async () => {
    render(<ConfigurationTab machine={machine()} onInstallCapability={vi.fn()} />);

    const row = await screen.findByTestId("machine-drift-node-bridge-provisioner");
    expect(row).toHaveTextContent("the privileged provisioning helper is not installed");
    expect(row).not.toHaveTextContent("helper_not_installed");
    expect(row.querySelector("button")).toBeNull();
  });

  it("re-applies from the profile row and reports which steps failed and why, at the top of the panel", async () => {
    apiMocks.reapplyConfiguration.mockResolvedValue({ targetId: "minimouse", result: { run: { runId: "apply-1" } } });
    apiMocks.getConfigurationApplyStatus.mockResolvedValue({
      status: ApplyRunState.PARTIALLY_APPLIED,
      steps: [
        { id: "tool:buf", name: "buf", state: ApplyStepState.APPLIED, error: "", remediation: "", errorCode: "" },
        {
          id: "scenario:secrets-manager",
          name: "secrets-manager",
          state: ApplyStepState.FAILED,
          error: '{"msg":"Scenario start failed","error":{"message":"start postgres: process environment inspection is not supported on this platform"}}',
          remediation: "",
          errorCode: "",
        },
      ],
    });
    render(<ConfigurationTab machine={machine()} />);

    fireEvent.click(await screen.findByTestId("machine-drift-reapply"));

    const panel = await screen.findByTestId("machine-configuration-apply");
    await waitFor(() => {
      expect(panel).toHaveAttribute("data-apply-outcome", "partial");
    });
    expect(screen.getByTestId("machine-configuration-apply-failed-scenario:secrets-manager")).toHaveTextContent(
      "start postgres: process environment inspection is not supported on this platform",
    );
    const container = screen.getByTestId("machine-configuration-panel");
    expect(container.firstElementChild).toBe(panel);
  });

  it("shows why a re-apply could not start", async () => {
    apiMocks.reapplyConfiguration.mockRejectedValue(new Error("resolve target onboarding endpoint: node offline"));
    render(<ConfigurationTab machine={machine()} />);

    fireEvent.click(await screen.findByTestId("machine-drift-reapply"));

    const panel = await screen.findByTestId("machine-configuration-apply");
    await waitFor(() => {
      expect(panel).toHaveAttribute("data-apply-phase", "error");
    });
    expect(panel).toHaveTextContent("resolve target onboarding endpoint: node offline");
  });
});
