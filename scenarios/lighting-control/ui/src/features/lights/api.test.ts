import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const clients = vi.hoisted(() => ({
  control: {
    getBackendStatus: vi.fn(),
    listControllers: vi.fn(),
    getDesiredState: vi.fn(),
    enrollController: vi.fn(),
    setManualOff: vi.fn(),
    clearManualOverride: vi.fn(),
    reconcile: vi.fn(),
    setZoneState: vi.fn(),
  },
  policy: {
    getPolicy: vi.fn(),
    setManualPreset: vi.fn(),
    clearManualPreset: vi.fn(),
    setPolicy: vi.fn(),
    evaluatePolicy: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: (service: { typeName: string }) =>
    service.typeName.endsWith("ControlService") ? clients.control : clients.policy,
}));

import { lightingApi, PresetKind } from "./api";

describe("lightingApi", () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("reads backend, inventory, desired state, and policy together into the desk model", async () => {
    const backend = { available: true, version: "1.1.0" };
    const controllers = [{ id: "gpu", available: true }];
    const zones = [{ controllerId: "gpu", id: "main" }];
    const manualOverride = { active: true, allEnrolled: false, controllerIds: [] };
    const effective = { preset: PresetKind.WORK };
    const config = { presets: [] };
    clients.control.getBackendStatus.mockResolvedValue({ backend });
    clients.control.listControllers.mockResolvedValue({ controllers });
    clients.control.getDesiredState.mockResolvedValue({ zones, manualOverride });
    clients.policy.getPolicy.mockResolvedValue({ effective, config });

    await expect(lightingApi.readDesk()).resolves.toEqual({
      backend,
      controllers,
      zones,
      manualOverride,
      effective,
      config,
    });

    expect(clients.control.getBackendStatus.mock.calls[0]?.[0]).toMatchObject({
      $typeName: "vrooli.lighting_control.v1.control.GetBackendStatusRequest",
    });
    expect(clients.control.listControllers.mock.calls[0]?.[0]).toMatchObject({ includeUnavailable: true });
    expect(clients.control.getDesiredState.mock.calls[0]?.[0]).toMatchObject({
      $typeName: "vrooli.lighting_control.v1.control.GetDesiredStateRequest",
    });
    expect(clients.policy.getPolicy.mock.calls[0]?.[0]).toMatchObject({
      $typeName: "vrooli.lighting_control.v1.policy.GetPolicyRequest",
    });
  });

  it("does not return a partial desk snapshot when one of its required reads fails", async () => {
    clients.control.getBackendStatus.mockResolvedValue({ backend: { available: true } });
    clients.control.listControllers.mockRejectedValue(new Error("inventory unavailable"));
    clients.control.getDesiredState.mockResolvedValue({ zones: [] });
    clients.policy.getPolicy.mockResolvedValue({ effective: {}, config: {} });

    await expect(lightingApi.readDesk()).rejects.toThrow("inventory unavailable");
    expect(clients.control.getDesiredState).toHaveBeenCalledOnce();
    expect(clients.policy.getPolicy).toHaveBeenCalledOnce();
  });

  it("projects manual Off over the base desired state for the desk", async () => {
    clients.control.getBackendStatus.mockResolvedValue({});
    clients.control.listControllers.mockResolvedValue({ controllers: [] });
    clients.control.getDesiredState.mockResolvedValue({
      zones: [{ desired: { controllerId: "gpu", zoneId: "main", desiredMode: { case: "staticColor", value: { color: { red: 1, green: 2, blue: 3 } } } } }],
      manualOverride: { active: true, allEnrolled: false, controllerIds: ["gpu"] },
    });
    clients.policy.getPolicy.mockResolvedValue({});

    const desk = await lightingApi.readDesk();
    expect(desk.zones[0]?.desired?.desiredMode.case).toBe("off");
    expect(desk.zones[0]?.desired?.controllerId).toBe("gpu");
  });

  it("maps desk actions to the control and policy request contracts", async () => {
    const config = { brightnessCapPercent: 35 };

    await lightingApi.enroll("gpu", true);
    await lightingApi.setOff();
    await lightingApi.clearOff();
    await lightingApi.reconcile();
    await lightingApi.choosePreset(PresetKind.HEALTH);
    await lightingApi.clearPreset();
    await lightingApi.savePolicy(config as never);
    await lightingApi.evaluatePolicy();

    expect(clients.control.enrollController.mock.calls[0]?.[0]).toMatchObject({ controllerId: "gpu", enrolled: true });
    expect(clients.control.setManualOff.mock.calls[0]?.[0]).toMatchObject({ allEnrolled: true });
    expect(clients.control.clearManualOverride.mock.calls[0]?.[0]).toMatchObject({
      $typeName: "vrooli.lighting_control.v1.control.ClearManualOverrideRequest",
    });
    expect(clients.control.reconcile.mock.calls[0]?.[0]).toMatchObject({ trigger: "operator_ui" });
    expect(clients.policy.setManualPreset.mock.calls[0]?.[0]).toMatchObject({ preset: PresetKind.HEALTH });
    expect(clients.policy.clearManualPreset.mock.calls[0]?.[0]).toMatchObject({
      $typeName: "vrooli.lighting_control.v1.policy.ClearManualPresetRequest",
    });
    expect(clients.policy.setPolicy.mock.calls[0]?.[0]).toMatchObject({ config });
    expect(clients.policy.evaluatePolicy.mock.calls[0]?.[0]).toMatchObject({ trigger: "operator_settings" });
  });

  it("builds an explicit static zone state with the selected color and brightness", async () => {
    await lightingApi.setStatic("gpu", "main", { red: 12, green: 34, blue: 56 }, 40);

    expect(clients.control.setZoneState.mock.calls[0]?.[0]).toMatchObject({
      desired: {
        controllerId: "gpu",
        zoneId: "main",
        desiredMode: {
          case: "staticColor",
          value: {
            color: { red: 12, green: 34, blue: 56 },
            brightnessPercent: 40,
          },
        },
      },
    });
  });
  it("builds the existing typed effect request without changing ownership or Off", async () => {
    await lightingApi.setEffect("gpu", "main", "Breathing", { red: 12, green: 34, blue: 56 }, 10);
    expect(clients.control.setZoneState.mock.calls[0]?.[0]).toMatchObject({ desired: {
      controllerId: "gpu", zoneId: "main", desiredMode: { case: "effect", value: {
        effectId: "Breathing", color: { red: 12, green: 34, blue: 56 }, brightnessPercent: 10,
      } },
    } });
    expect(clients.control.clearManualOverride).not.toHaveBeenCalled();
    expect(clients.policy.setManualPreset).not.toHaveBeenCalled();
  });

});
