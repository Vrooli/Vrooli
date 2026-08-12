import { beforeEach, describe, expect, it } from "vitest";
import { deviceIdentity, setDeviceLabel } from "./deviceIdentity";
describe("deviceIdentity", () => {
    beforeEach(() => localStorage.clear());
    it("keeps the generated identifier across reload-like reads", () => {
        const first = deviceIdentity();
        expect(deviceIdentity().id).toBe(first.id);
        setDeviceLabel("Desk");
        expect(deviceIdentity().label).toBe("Desk");
    });
});
