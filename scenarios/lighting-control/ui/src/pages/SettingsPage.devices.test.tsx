import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { InstallationSource, LightingRuntimeStatusSchema, LightingSelectionSchema, LightingDeviceCandidateSchema, LightingDeviceDiscoverySchema, LightingDevicePlanSchema, LightingDeviceAccessResultSchema } from "@vrooli/proto-types/lighting-control/v1/setup/setup_pb";
import { PolicyConfigSchema } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";
import { lightingSetupApi } from "../features/lights/setup";
import { lightingApi } from "../features/lights/api";
import { sessionApi } from "../api/session";
import { strings } from "../consts/strings";
import { renderWithProviders } from "../test-utils";
import { SettingsPage } from "./SettingsPage";

const copy = strings.lightDesk.settingsUi;
const ownerPolicy = "owner-policy";
const board = create(LightingDeviceCandidateSchema, { id: "board-usb", label: "Fixture board USB lighting", kind: "usb-hid", detail: "Exact selected interface" });
const gpu = create(LightingDeviceCandidateSchema, { id: "gpu-pci", label: "Fixture GPU lighting", kind: "pci-i2c" });
const selection = create(LightingSelectionSchema, { source: InstallationSource.MANAGED, version: "1.1", sha256: "a".repeat(64) });
const disabled = () => create(LightingRuntimeStatusSchema, { phase: "disabled", state: "disabled", pinnedSelection: selection });
const plan = () => create(LightingDevicePlanSchema, { id: "reviewed-plan", devices: [board], changes: ["Fixture narrow board grant"], warnings: ["Fixture scope warning"], requiresPrivilege: true });
const renderSettings = () => renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><SettingsPage /></MemoryRouter>, { withoutRouter: true });

async function discoverAndPlan() {
  fireEvent.click(await screen.findByRole("button", { name: copy.discoverDevices }));
  fireEvent.click(await screen.findByRole("checkbox", { name: new RegExp(board.label) }));
  fireEvent.click(screen.getByRole("button", { name: copy.reviewDevicePlan }));
  return screen.findByRole("button", { name: copy.applyDevicePlan });
}

describe("Settings maintained device access", () => {
  beforeEach(() => {
    vi.spyOn(sessionApi, "status").mockResolvedValue({ authenticated: false });
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined, config: create(PolicyConfigSchema, {}) });
    vi.spyOn(lightingSetupApi, "status").mockResolvedValue(disabled());
    vi.spyOn(lightingSetupApi, "devices").mockResolvedValue(create(LightingDeviceDiscoverySchema, { candidates: [board, gpu], state: "available" }));
    vi.spyOn(lightingSetupApi, "planDevices").mockResolvedValue(plan());
  });
  afterEach(() => vi.restoreAllMocks());

  it("opens without device discovery, planning or provisioning", async () => {
    const apply = vi.spyOn(lightingSetupApi, "applyDevices");
    const devices = vi.spyOn(lightingSetupApi, "devices");
    const planDevices = vi.spyOn(lightingSetupApi, "planDevices");
    renderSettings();
    expect(await screen.findByRole("button", { name: copy.discoverDevices })).toBeEnabled();
    expect(devices).not.toHaveBeenCalled();
    expect(planDevices).not.toHaveBeenCalled();
    expect(apply).not.toHaveBeenCalled();
  });

  it("reviews only selected devices before explicit apply and requires fresh setup", async () => {
    vi.spyOn(lightingSetupApi, "status").mockResolvedValue(create(LightingRuntimeStatusSchema, { ...disabled(), pinnedSelection: create(LightingSelectionSchema, { ...selection, devicePolicyId: ownerPolicy }) }));
    const apply = vi.spyOn(lightingSetupApi, "applyDevices").mockResolvedValue(create(LightingDeviceAccessResultSchema, { planId: "reviewed-plan", state: "applied", devicePolicyId: ownerPolicy }));
    const planDevices = vi.spyOn(lightingSetupApi, "planDevices");
    const setup = vi.spyOn(lightingSetupApi, "setup");
    const enable = vi.spyOn(lightingSetupApi, "enable");
    renderSettings();
    const applyButton = await discoverAndPlan();
    expect(planDevices.mock.calls[0]?.[0]).toEqual([board.id]);
    expect(screen.getByText(plan().changes[0] ?? "")).toBeInTheDocument();
    expect(screen.getByText(plan().warnings[0] ?? "")).toBeInTheDocument();
    expect(apply).not.toHaveBeenCalled();
    fireEvent.click(applyButton);
    await waitFor(() => expect(apply).toHaveBeenCalledWith("reviewed-plan"));
    expect(await screen.findByText(ownerPolicy)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: copy.enableLighting })).toBeDisabled();
    expect(setup).not.toHaveBeenCalled(); expect(enable).not.toHaveBeenCalled();
  });

  it("invalidates a preview when the selected device set changes", async () => {
    const apply = vi.spyOn(lightingSetupApi, "applyDevices");
    renderSettings();
    await discoverAndPlan();
    fireEvent.click(screen.getByRole("checkbox", { name: new RegExp(gpu.label) }));
    expect(screen.queryByRole("button", { name: copy.applyDevicePlan })).not.toBeInTheDocument();
    expect(apply).not.toHaveBeenCalled();
  });

  it("shows the owner's local privilege instruction without claiming applied access", async () => {
    const instruction = "Fixture owner privileged operation";
    vi.spyOn(lightingSetupApi, "applyDevices").mockResolvedValue(create(LightingDeviceAccessResultSchema, { planId: "reviewed-plan", state: "needs_sudo", needsSudo: true, message: instruction }));
    renderSettings();
    fireEvent.click(await discoverAndPlan());
    expect(await screen.findByText(instruction)).toBeInTheDocument();
    expect(screen.getByText(copy.deviceNeedsSudo)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: copy.rollbackDevicePlan })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: copy.enableLighting })).toBeDisabled();
  });

  it("recovers rollback authority from retained status without discovering devices", async () => {
    vi.spyOn(lightingSetupApi, "status").mockResolvedValue(create(LightingRuntimeStatusSchema, { ...disabled(), deviceAccessPlanId: "retained-applied-plan" }));
    const devices = vi.spyOn(lightingSetupApi, "devices");
    const rollback = vi.spyOn(lightingSetupApi, "rollbackDevices").mockResolvedValue(create(LightingDeviceAccessResultSchema, { planId: "retained-applied-plan", state: "rolled_back" }));
    renderSettings();
    fireEvent.click(await screen.findByRole("button", { name: copy.rollbackDevicePlan }));
    await waitFor(() => expect(rollback).toHaveBeenCalledWith("retained-applied-plan"));
    expect(devices).not.toHaveBeenCalled();
  });

  it("keeps an uncertain apply on the exact retained plan for a deliberate retry", async () => {
    const apply = vi.spyOn(lightingSetupApi, "applyDevices").mockRejectedValueOnce(new Error("Fixture response lost")).mockResolvedValue(create(LightingDeviceAccessResultSchema, { planId: "reviewed-plan", state: "applied", devicePolicyId: ownerPolicy }));
    renderSettings();
    fireEvent.click(await discoverAndPlan());
    expect(await screen.findByRole("alert")).toHaveTextContent("Fixture response lost");
    const retry = await screen.findByRole("button", { name: copy.retryDevices });
    expect(screen.getByRole("button", { name: copy.applyDevicePlan })).toBeDisabled();
    fireEvent.click(retry);
    await waitFor(() => expect(screen.getByRole("button", { name: copy.applyDevicePlan })).toBeEnabled());
    fireEvent.click(screen.getByRole("button", { name: copy.applyDevicePlan }));
    await waitFor(() => expect(apply).toHaveBeenCalledTimes(2));
    expect(apply.mock.calls.map(([id]) => id)).toEqual(["reviewed-plan", "reviewed-plan"]);
  });
  it("retains partial error details as recovery evidence and blocks fresh setup", async () => {
    const ownerMessage = "Fixture permission state remains pending";
    const failure = new ConnectError("Fixture apply incomplete", Code.FailedPrecondition, undefined, [{ desc: LightingDeviceAccessResultSchema, value: { planId: "reviewed-plan", state: "pending", message: ownerMessage } }]);
    vi.spyOn(lightingSetupApi, "applyDevices").mockRejectedValue(failure);
    const rollback = vi.spyOn(lightingSetupApi, "rollbackDevices").mockResolvedValue(create(LightingDeviceAccessResultSchema, { planId: "reviewed-plan", state: "rolled_back" }));
    renderSettings();
    fireEvent.click(await discoverAndPlan());
    expect(await screen.findByText(ownerMessage)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: copy.prepareSetup })).toBeDisabled();
    expect(screen.getByRole("button", { name: copy.enableLighting })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: copy.retryDevices }));
    await waitFor(() => expect(screen.getByRole("button", { name: copy.rollbackDevicePlan })).toBeEnabled());
    expect(screen.getByRole("button", { name: copy.reviewDevicePlan })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: copy.rollbackDevicePlan }));
    await waitFor(() => expect(rollback).toHaveBeenCalledWith("reviewed-plan"));
    await waitFor(() => expect(screen.getByRole("button", { name: copy.prepareSetup })).toBeEnabled());
  });

  it("restores persisted pending recovery after reload without permitting fresh setup", async () => {
    vi.spyOn(lightingSetupApi, "status").mockResolvedValue(create(LightingRuntimeStatusSchema, { ...disabled(), deviceAccessPlanId: "retained-pending-plan", deviceAccessPending: true }));
    const rollback = vi.spyOn(lightingSetupApi, "rollbackDevices").mockResolvedValue(create(LightingDeviceAccessResultSchema, { planId: "retained-pending-plan", state: "rolled_back" }));
    renderSettings();
    const recover = await screen.findByRole("button", { name: copy.rollbackDevicePlan });
    await screen.findByLabelText(copy.installationSource);
    expect(screen.getByRole("button", { name: copy.prepareSetup })).toBeDisabled();
    expect(screen.getByRole("button", { name: copy.enableLighting })).toBeDisabled();
    expect(recover).toBeEnabled();
    fireEvent.click(recover);
    await waitFor(() => expect(rollback).toHaveBeenCalledWith("retained-pending-plan"));
  });

  it("does not turn a failed plan lookup into rollback authority", async () => {
    vi.spyOn(lightingSetupApi, "applyDevices").mockRejectedValue(new ConnectError("Fixture plan absent", Code.FailedPrecondition));
    renderSettings();
    fireEvent.click(await discoverAndPlan());
    expect(await screen.findByRole("alert")).toHaveTextContent("Fixture plan absent");
    expect(screen.queryByRole("button", { name: copy.rollbackDevicePlan })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: copy.ownerRefresh }));
    await waitFor(() => expect(screen.getByRole("button", { name: copy.prepareSetup })).toBeEnabled());
  });

});
