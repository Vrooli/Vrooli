import { expect } from "chai";
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
            expect(result).to.equal("");
        });

        it("should return non-WindowsKey values as-is", () => {
            expect(keyModule.keyComboToString("Shift")).to.equal("Shift");
            expect(keyModule.keyComboToString("A")).to.equal("A");
        });

        it("should keep WindowsKey values unchanged", () => {
            expect(keyModule.keyComboToString("Ctrl")).to.equal("Ctrl");
            expect(keyModule.keyComboToString("Alt")).to.equal("Alt");
            expect(keyModule.keyComboToString("Enter")).to.equal("Enter");
        });

        it("should join multiple keys with \" + \"", () => {
            expect(keyModule.keyComboToString("Ctrl", "A")).to.equal("Ctrl + A");
            expect(keyModule.keyComboToString("Shift", "Tab")).to.equal("Shift + Tab");
        });

        it("should handle mixed keys without conversion", () => {
            expect(keyModule.keyComboToString("Ctrl", "Shift", "B")).to.equal("Ctrl + Shift + B");
        });

        it("should handle arrow keys and number keys", () => {
            expect(keyModule.keyComboToString("ArrowUp", "1")).to.equal("ArrowUp + 1");
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
            expect(keyModule.keyComboToString("Ctrl")).to.equal(keyModule.MacKeyFromWindows.Ctrl);
            expect(keyModule.keyComboToString("Alt")).to.equal(keyModule.MacKeyFromWindows.Alt);
            expect(keyModule.keyComboToString("Enter")).to.equal(keyModule.MacKeyFromWindows.Enter);
        });

        it("should leave non-WindowsKey values unchanged", () => {
            expect(keyModule.keyComboToString("Shift")).to.equal("Shift");
            expect(keyModule.keyComboToString("A")).to.equal("A");
        });

        it("should join multiple keys with proper conversion", () => {
            expect(keyModule.keyComboToString("Ctrl", "Shift", "A")).to.equal(
                `${keyModule.MacKeyFromWindows.Ctrl} + Shift + A`,
            );
            expect(keyModule.keyComboToString("Alt", "Tab")).to.equal(
                `${keyModule.MacKeyFromWindows.Alt} + Tab`,
            );
        });

        it("should handle a mixed key combination correctly", () => {
            const result = keyModule.keyComboToString("Ctrl", "Alt", "Delete");
            // Only "Ctrl" and "Alt" are converted; "Delete" remains unchanged.
            expect(result).to.equal(
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
            expect(keyModule.keyComboToString("Ctrl", "A")).to.equal("Ctrl + A");
        });
    });
});
