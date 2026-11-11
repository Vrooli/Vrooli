import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const clients = vi.hoisted(() => ({
  control: {
    getBackendStatus: vi.fn(),
    listControllers: vi.fn(),
    getDesiredState: vi.fn(),
    enrollController: vi.fn(),
    configureZone: vi.fn(),
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

it("sends explicit LED length through the typed public configuration method", async () => {
 clients.control.configureZone.mockResolvedValue({ zone: { configuredLedCount: 24 } });
 await lightingApi.configureZone("terminal", "header", 24);
 expect(clients.control.configureZone).toHaveBeenCalledWith(expect.objectContaining({ controllerId: "terminal", zoneId: "header", ledCount: 24 }));
});

it("keeps saved wiring and manual Off inspectable without upgrading a failed inventory probe", async () => {
  const { BackendState } = await import("@vrooli/proto-types/lighting-control/v1/control/control_pb");
  const unavailable = { state: BackendState.UNAVAILABLE, detail: "OpenRGB SDK transport failure" };
  const controllers = [{ controllerId: "saved", enrolled: true, zones: [{ zoneId: "header", ledCount: 24 }] }];
  clients.control.getBackendStatus.mockResolvedValue({ backend: { state: BackendState.READY } });
  clients.control.listControllers.mockResolvedValue({ backend: unavailable, controllers });
  clients.control.getDesiredState.mockResolvedValue({ zones: [], manualOverride: { active: true, allEnrolled: true, controllerIds: [] } });
  clients.policy.getPolicy.mockResolvedValue({});
  const snapshot = await lightingApi.readDesk();
  expect(snapshot.backend).toEqual(unavailable);
  expect(snapshot.controllers).toEqual(controllers);
  expect(snapshot.manualOverride?.active).toBe(true);
  expect(clients.control.listControllers).toHaveBeenCalledWith(expect.objectContaining({ includeUnavailable: true }));
});

it("does not upgrade an unavailable backend probe using an earlier ready inventory", async () => {
  const { BackendState } = await import("@vrooli/proto-types/lighting-control/v1/control/control_pb");
  const stopped = { state: BackendState.STOPPED };
  clients.control.getBackendStatus.mockResolvedValue({ backend: stopped });
  clients.control.listControllers.mockResolvedValue({ backend: { state: BackendState.READY }, controllers: [] });
  clients.control.getDesiredState.mockResolvedValue({ zones: [] });
  clients.policy.getPolicy.mockResolvedValue({});
  expect((await lightingApi.readDesk()).backend).toEqual(stopped);
});


it("projects scoped Off onto every selected controller zone without altering other manual colors", async () => {
  const base = (controllerId: string, zoneId: string) => ({ desired: { controllerId, zoneId, desiredMode: { case: "staticColor", value: { color: { red: 7, green: 8, blue: 9 } } } } });
  clients.control.getBackendStatus.mockResolvedValue({});
  clients.control.listControllers.mockResolvedValue({ controllers: [] });
  clients.policy.getPolicy.mockResolvedValue({});
  clients.control.getDesiredState.mockResolvedValue({ zones: [base("a", "one"), base("a", "two"), base("b", "one")], manualOverride: { active: true, allEnrolled: false, controllerIds: ["a"] } });
  const snapshot = await lightingApi.readDesk();
  expect(snapshot.zones.map((zone) => zone.desired?.desiredMode.case)).toEqual(["off", "off", "staticColor"]);
  expect(snapshot.zones[2]?.desired?.desiredMode).toMatchObject({ value: { color: { red: 7, green: 8, blue: 9 } } });
});
