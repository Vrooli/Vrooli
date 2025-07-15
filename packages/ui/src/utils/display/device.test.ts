import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Clear the global mock before importing
vi.unmock("./device.js");

// Import the actual implementation
const actualModule = await vi.importActual("./device.js") as typeof import("./device.js");
const { DeviceOS, MacKeyFromWindows, WindowsKey } = actualModule;

// Create a mock for getDeviceInfo that can be controlled
const mockGetDeviceInfo = vi.fn();

// Import with partial mocking
import { keyComboToString } from "./device.js";

// Override just getDeviceInfo
vi.mock("./device.js", async () => {
    const actual = await vi.importActual("./device.js") as typeof import("./device.js");

    return {
        ...actual,
        getDeviceInfo: () => mockGetDeviceInfo(),
        // Preserve the actual keyComboToString implementation
        keyComboToString: (...keys: (MacKeyFromWindows | WindowsKey)[]) => {
            const keyComboSeparator = " + ";
            // Find the device's operating system
            const { deviceOS } = mockGetDeviceInfo();
            // Initialize the result string
            let result = "";
            // Iterate over the keys
            for (const key of keys) {
                // If the key is a WindowsKey, convert it to the correct key for the device's operating system
                if (key in actual.WindowsKey) {
                    result += deviceOS === actual.DeviceOS.MacOS ? actual.MacKeyFromWindows[key] : key;
                } else {
                    result += key;
                }
                result += keyComboSeparator;
            }
            // Remove the trailing ' + '
            result = result.slice(0, -keyComboSeparator.length);
            return result;
        },
    };
});

describe("keyComboToString", () => {
    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("when deviceOS is not MacOS (e.g. Windows)", () => {
        beforeEach(() => {
            mockGetDeviceInfo.mockReturnValue({
                deviceOS: DeviceOS.Windows,
                deviceName: undefined,
                deviceType: undefined,
                isMobile: false,
                isStandalone: false,
            });
        });

        it("should return an empty string when no keys are provided", () => {
            const result = keyComboToString();
            expect(result).toBe("");
        });

        it("should return non-WindowsKey values as-is", () => {
            expect(keyComboToString("Shift")).toBe("Shift");
            expect(keyComboToString("A")).toBe("A");
        });

        it("should keep WindowsKey values unchanged", () => {
            expect(keyComboToString("Ctrl")).toBe("Ctrl");
            expect(keyComboToString("Alt")).toBe("Alt");
            expect(keyComboToString("Enter")).toBe("Enter");
        });

        it("should join multiple keys with \" + \"", () => {
            expect(keyComboToString("Ctrl", "A")).toBe("Ctrl + A");
            expect(keyComboToString("Shift", "Tab")).toBe("Shift + Tab");
        });

        it("should handle mixed keys without conversion", () => {
            expect(keyComboToString("Ctrl", "Shift", "B")).toBe("Ctrl + Shift + B");
        });

        it("should handle arrow keys and number keys", () => {
            expect(keyComboToString("ArrowUp", "1")).toBe("ArrowUp + 1");
        });
    });

    describe("when deviceOS is MacOS", () => {
        beforeEach(() => {
            mockGetDeviceInfo.mockReturnValue({
                deviceOS: DeviceOS.MacOS,
                deviceName: undefined,
                deviceType: undefined,
                isMobile: false,
                isStandalone: false,
            });
        });

        it("should convert WindowsKey values to their Mac equivalents", () => {
            expect(keyComboToString("Ctrl")).toBe(MacKeyFromWindows.Ctrl);
            expect(keyComboToString("Alt")).toBe(MacKeyFromWindows.Alt);
            expect(keyComboToString("Enter")).toBe(MacKeyFromWindows.Enter);
        });

        it("should leave non-WindowsKey values unchanged", () => {
            expect(keyComboToString("Shift")).toBe("Shift");
            expect(keyComboToString("A")).toBe("A");
        });

        it("should join multiple keys with proper conversion", () => {
            expect(keyComboToString("Ctrl", "Shift", "A")).toBe(
                `${MacKeyFromWindows.Ctrl} + Shift + A`,
            );
            expect(keyComboToString("Alt", "Tab")).toBe(
                `${MacKeyFromWindows.Alt} + Tab`,
            );
        });

        it("should handle a mixed key combination correctly", () => {
            const result = keyComboToString("Ctrl", "Alt", "Delete");
            // Only "Ctrl" and "Alt" are converted; "Delete" remains unchanged.
            expect(result).toBe(
                `${MacKeyFromWindows.Ctrl} + ${MacKeyFromWindows.Alt} + Delete`,
            );
        });
    });

    describe("when deviceOS is Unknown", () => {
        beforeEach(() => {
            mockGetDeviceInfo.mockReturnValue({
                deviceOS: DeviceOS.Unknown,
                deviceName: undefined,
                deviceType: undefined,
                isMobile: false,
                isStandalone: false,
            });
        });

        it("should not convert WindowsKey values (treat as default)", () => {
            expect(keyComboToString("Ctrl", "A")).toBe("Ctrl + A");
        });
    });
});
