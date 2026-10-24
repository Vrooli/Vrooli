import { describe, expect, it } from "vitest";
import { bundledEnvironmentBase } from "../bundled-environment";

describe("bundledEnvironmentBase", () => {
    it("preserves launch attribution and excludes host agent routing", () => {
        const env = bundledEnvironmentBase({
            HOME: "/host/home",
            PATH: "/host/bin",
            S2D_PROCESS_SESSION: "launch-123",
            VROOLI_ROOT: "/host/vrooli",
            VROOLI_AGENT_SESSION: "host-agent-456",
        });

        expect(env).toEqual({
            HOME: "/host/home",
            PATH: "/host/bin",
            S2D_PROCESS_SESSION: "launch-123",
        });
        expect(env).not.toHaveProperty("VROOLI_ROOT");
        expect(env).not.toHaveProperty("VROOLI_AGENT_SESSION");
    });
});
