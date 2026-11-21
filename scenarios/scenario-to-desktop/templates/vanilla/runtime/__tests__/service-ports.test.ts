import { describe, expect, it } from "vitest";
import { selectBundledAPIPort } from "../service-ports";

describe("selectBundledAPIPort", () => {
    it("selects the root API ahead of alphabetically earlier peer APIs", () => {
        const services = {
            "ai-gateway--api": { api: 19101 },
            "audio-tools--api": { api: 19102 },
            api: { api: 19100 },
            ui: { ui: 20003 },
        };

        expect(selectBundledAPIPort(services, "ui")).toBe(19100);
    });

    it("selects a named root backend when the root service is not named api", () => {
        const services = {
            "billing-api": { http: 19110 },
            "remote--api": { api: 19111 },
            ui: { ui: 20003 },
        };

        expect(selectBundledAPIPort(services, "ui")).toBe(19110);
    });

    it("does not use a peer API as the root app endpoint", () => {
        expect(selectBundledAPIPort({ "ai-gateway--api": { api: 19101 }, ui: { ui: 20003 } }, "ui"))
            .toBeUndefined();
    });
});
