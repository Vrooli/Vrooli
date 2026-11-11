import { create } from "@bufbuild/protobuf";
import { describe, expect, it, vi } from "vitest";
import { InstallationSource, LightingSelectionSchema } from "@vrooli/proto-types/lighting-control/v1/setup/setup_pb";

const client = vi.hoisted(() => ({ getSetupDevices: vi.fn(), planLightingDevices: vi.fn(), applyLightingDevices: vi.fn(), rollbackLightingDevices: vi.fn(), getSetupStatus: vi.fn(), setupLighting: vi.fn(), enableLighting: vi.fn(), disableLighting: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => client }));
import { lightingSetupApi } from "./setup";

describe("lightingSetupApi", () => {
  it("uses the authenticated typed transport with bounded owner requests", async () => {
    const selection = create(LightingSelectionSchema, { source: InstallationSource.MANAGED, version: "1.1", sha256: "a".repeat(64), devicePolicyId: "policy" });
    await lightingSetupApi.status();
    await lightingSetupApi.setup(selection);
    await lightingSetupApi.enable(selection);
    await lightingSetupApi.disable();
    expect(client.getSetupStatus).toHaveBeenCalledWith({ $typeName: "vrooli.lighting_control.v1.setup.GetSetupStatusRequest" });
    expect(client.setupLighting.mock.calls[0]?.[0].selection).toEqual(selection);
    expect(client.enableLighting.mock.calls[0]?.[0].selection).toEqual(selection);
    expect(client.disableLighting).toHaveBeenCalledWith({ $typeName: "vrooli.lighting_control.v1.setup.DisableLightingRequest" });
  });
  it("passes exact discovered device IDs and immutable plan references", async () => {
    await lightingSetupApi.devices();
    await lightingSetupApi.planDevices(["board-usb"]);
    await lightingSetupApi.applyDevices("reviewed-plan");
    await lightingSetupApi.rollbackDevices("applied-plan");
    expect(client.getSetupDevices).toHaveBeenCalledOnce();
    expect(client.planLightingDevices.mock.calls[0]?.[0].deviceIds).toEqual(["board-usb"]);
    expect(client.applyLightingDevices.mock.calls[0]?.[0].planId).toBe("reviewed-plan");
    expect(client.rollbackLightingDevices.mock.calls[0]?.[0].planId).toBe("applied-plan");
  });

});
