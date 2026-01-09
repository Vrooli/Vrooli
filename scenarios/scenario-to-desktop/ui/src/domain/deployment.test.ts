import { describe, expect, it } from "vitest";
import {
  DEFAULT_SERVER_TYPE,
  decideConnection,
  type DeploymentMode,
  type ServerType
} from "./deployment";

describe("decideConnection", () => {
  const evaluate = (mode: DeploymentMode, serverType: ServerType) =>
    decideConnection(mode, serverType);

  it("requires bundle manifests and disables auto-manage in bundled mode", () => {
    const decision = evaluate("bundled", "external");

    expect(decision.kind).toBe("bundled-runtime");
    expect(decision.requiresBundleManifest).toBe(true);
    expect(decision.requiresProxyUrl).toBe(false);
    expect(decision.effectiveServerType).toBe(DEFAULT_SERVER_TYPE);
    expect(decision.allowsAutoManageTier1).toBe(false);
  });

  it("routes thin clients to remote servers and enforces proxy URLs", () => {
    const decision = evaluate("external-server", "external");

    expect(decision.kind).toBe("remote-server");
    expect(decision.requiresProxyUrl).toBe(true);
    expect(decision.requiresBundleManifest).toBe(false);
    expect(decision.effectiveServerType).toBe("external");
    expect(decision.allowsAutoManageTier1).toBe(true);
  });

  it("keeps embedded/server binaries local without proxy or manifest requirements", () => {
    const decision = evaluate("external-server", "node");

    expect(decision.kind).toBe("local-embedded");
    expect(decision.requiresProxyUrl).toBe(false);
    expect(decision.requiresBundleManifest).toBe(false);
    expect(decision.effectiveServerType).toBe("node");
    expect(decision.allowsAutoManageTier1).toBe(true);
  });
});
