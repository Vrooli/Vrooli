import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import sinon from "sinon";
import * as keyModule from "./device.js";

describe("keyComboToString", () => {
    let getDeviceInfoStub: sinon.SinonStub;

    afterEach(() => {
        sinon.restore();
    });

    context("when deviceOS is not MacOS (e.g. Windows)", () => {
        beforeEach(() => {
            getDeviceInfoStub = sinon
                .stub(keyModule, "getDeviceInfo")
                .returns({
                    deviceOS: keyModule.DeviceOS.Windows,
                    deviceName: undefined,
                    deviceType: undefined,
                    isMobile: false,
                    isStandalone: false,
                });
        });

        it("should return an empty string when no keys are provided", () => {
            const result = keyModule.keyComboToString();
            expect(result).toBe("");
        });

        it("should return non-WindowsKey values as-is", () => {
            expect(keyModule.keyComboToString("Shift")).toBe("Shift");
            expect(keyModule.keyComboToString("A")).toBe("A");
        });

        it("should keep WindowsKey values unchanged", () => {
            expect(keyModule.keyComboToString("Ctrl")).toBe("Ctrl");
            expect(keyModule.keyComboToString("Alt")).toBe("Alt");
            expect(keyModule.keyComboToString("Enter")).toBe("Enter");
        });

        it("should join multiple keys with \" + \"", () => {
            expect(keyModule.keyComboToString("Ctrl", "A")).toBe("Ctrl + A");
            expect(keyModule.keyComboToString("Shift", "Tab")).toBe("Shift + Tab");
        });

        it("should handle mixed keys without conversion", () => {
            expect(keyModule.keyComboToString("Ctrl", "Shift", "B")).toBe("Ctrl + Shift + B");
        });

        it("should handle arrow keys and number keys", () => {
            expect(keyModule.keyComboToString("ArrowUp", "1")).toBe("ArrowUp + 1");
        });
    });

    context("when deviceOS is MacOS", () => {
        beforeEach(() => {
            getDeviceInfoStub = sinon
                .stub(keyModule, "getDeviceInfo")
                .returns({
                    deviceOS: keyModule.DeviceOS.MacOS,
                    deviceName: undefined,
                    deviceType: undefined,
                    isMobile: false,
                    isStandalone: false,
                });
        });

        it("should convert WindowsKey values to their Mac equivalents", () => {
            expect(keyModule.keyComboToString("Ctrl")).toBe(keyModule.MacKeyFromWindows.Ctrl);
            expect(keyModule.keyComboToString("Alt")).toBe(keyModule.MacKeyFromWindows.Alt);
            expect(keyModule.keyComboToString("Enter")).toBe(keyModule.MacKeyFromWindows.Enter);
        });

        it("should leave non-WindowsKey values unchanged", () => {
            expect(keyModule.keyComboToString("Shift")).toBe("Shift");
            expect(keyModule.keyComboToString("A")).toBe("A");
        });

        it("should join multiple keys with proper conversion", () => {
            expect(keyModule.keyComboToString("Ctrl", "Shift", "A")).toBe(
                `${keyModule.MacKeyFromWindows.Ctrl} + Shift + A`,
            );
            expect(keyModule.keyComboToString("Alt", "Tab")).toBe(
                `${keyModule.MacKeyFromWindows.Alt} + Tab`,
            );
        });

        it("should handle a mixed key combination correctly", () => {
            const result = keyModule.keyComboToString("Ctrl", "Alt", "Delete");
            // Only "Ctrl" and "Alt" are converted; "Delete" remains unchanged.
            expect(result).toBe(
                `${keyModule.MacKeyFromWindows.Ctrl} + ${keyModule.MacKeyFromWindows.Alt} + Delete`,
            );
        });
    });

    context("when deviceOS is Unknown", () => {
        beforeEach(() => {
            getDeviceInfoStub = sinon
                .stub(keyModule, "getDeviceInfo")
                .returns({
                    deviceOS: keyModule.DeviceOS.Unknown,
                    deviceName: undefined,
                    deviceType: undefined,
                    isMobile: false,
                    isStandalone: false,
                });
        });

        it("should not convert WindowsKey values (treat as default)", () => {
            expect(keyModule.keyComboToString("Ctrl", "A")).toBe("Ctrl + A");
        });
    });
});
