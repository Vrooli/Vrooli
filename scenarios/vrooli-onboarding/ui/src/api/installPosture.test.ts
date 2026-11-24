import { beforeEach, describe, expect, it, vi } from "vitest";
import { readInstallPostureFromRegistry, writeUpdateControlThroughRegistry } from "./installPostureContract";

const rpc = vi.hoisted(() => ({ read: vi.fn(), list: vi.fn(), write: vi.fn() }));
const client = { readPermission: rpc.read, listPermissions: rpc.list, writePermission: rpc.write };

describe("install posture registry client", () => {
  beforeEach(() => vi.clearAllMocks());

  it("reads each posture field from its owning registry provider", async () => {
    rpc.read.mockImplementation(async ({ permissionId }: { permissionId: string }) => permissionId.endsWith("update-control")
      ? { revision: "operator-r4", valueJson: new TextEncoder().encode(`{"value":"guard","supported_values":["observe","guard"]}`) }
      : permissionId.endsWith("trust-posture")
        ? { revision: "operator-r4", valueJson: new TextEncoder().encode(`{"value":"hosted","supported_values":["personal","shared","hosted"]}`) }
        : { revision: "enrollment-r2", valueJson: new TextEncoder().encode(`{"value":"shared","supported_values":["personal","shared","hosted"]}`) });
    rpc.list.mockResolvedValue({ permissions: [{ id: "onboarding/install-posture/session-mode" }] });
    await expect(readInstallPostureFromRegistry(client)).resolves.toEqual({ updateControl: "guard", updateControlRevision: "operator-r4", trustPosture: "hosted", sessionMode: "shared", sessionModeRevision: "enrollment-r2" });
    expect(rpc.read).toHaveBeenCalledWith({ permissionId: "onboarding/install-posture/update-control", owner: "vrooli-onboarding" });
    expect(rpc.read).toHaveBeenCalledWith({ permissionId: "onboarding/install-posture/trust-posture", owner: "vrooli-onboarding" });
    expect(rpc.list).toHaveBeenCalledWith({ owner: "api-core/operatorsession" });
    expect(rpc.read).toHaveBeenCalledWith({ permissionId: "onboarding/install-posture/session-mode", owner: "api-core/operatorsession" });
  });

  it("writes only the typed update-control value with a caller revision and receipt key", async () => {
    rpc.write.mockResolvedValue({ receipt: { id: "receipt-1" } });
    await writeUpdateControlThroughRegistry(client, "observe", "operator-r5", "request-1");
    expect(rpc.write).toHaveBeenCalledWith(expect.objectContaining({
      permissionId: "onboarding/install-posture/update-control", owner: "vrooli-onboarding",
      expectedRevision: "operator-r5", idempotencyKey: "request-1",
      valueJson: new TextEncoder().encode(`{"value":"observe"}`),
    }));
  });
});
