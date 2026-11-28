import { describe, expect, it } from "vitest";
import { bundledEnvironmentBase } from "../bundled-environment";

describe("bundledEnvironmentBase", () => {
    it("passes the app endpoint to the embedded Gateway without taking host routing", () => {
        const env = bundledEnvironmentBase({ LPBS_AI_DOOR_URL: "https://host.invalid" }, "https://app.example.test");
        expect(env.LPBS_AI_DOOR_URL).toBe("https://app.example.test");
    });

    it("preserves the selected native credential session and network proxy", () => {
        const env = bundledEnvironmentBase({ DBUS_SESSION_BUS_ADDRESS: "unix:path=/session/bus", XDG_RUNTIME_DIR: "/session", HTTPS_PROXY: "http://127.0.0.1:9000", NO_PROXY: "localhost,127.0.0.1" });
        expect(env.DBUS_SESSION_BUS_ADDRESS).toBe("unix:path=/session/bus");
        expect(env.XDG_RUNTIME_DIR).toBe("/session");
        expect(env.HTTPS_PROXY).toBe("http://127.0.0.1:9000");
        expect(env.NO_PROXY).toBe("localhost,127.0.0.1");
    });

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
