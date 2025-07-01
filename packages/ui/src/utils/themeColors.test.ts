// AI_CHECK: TEST_COVERAGE=1,TEST_QUALITY=1 | LAST: 2025-06-19
import { createTheme } from "@mui/material/styles";
import { describe, it, expect } from "vitest";
import { resolveThemeColor } from "./themeColors.js";

describe("resolveThemeColor", () => {
    const theme = createTheme({
        palette: {
            primary: {
                main: "#1976d2",
                light: "#42a5f5",
                dark: "#1565c0",
            },
            secondary: {
                main: "#dc004e",
                light: "#f48fb1",
                dark: "#880e4f",
            },
            text: {
                primary: "rgba(0, 0, 0, 0.87)",
                secondary: "rgba(0, 0, 0, 0.6)",
            },
            grey: {
                50: "#fafafa",
                100: "#f5f5f5",
                500: "#9e9e9e",
                900: "#212121",
            },
        },
    });

    it("returns 'currentColor' for null or undefined color", () => {
        expect(resolveThemeColor(theme, null)).toBe("currentColor");
        expect(resolveThemeColor(theme, undefined)).toBe("currentColor");
        expect(resolveThemeColor(theme, "")).toBe("currentColor");
    });

    it("resolves simple theme color paths", () => {
        expect(resolveThemeColor(theme, "primary")).toBe("#1976d2"); // Should get main color
        expect(resolveThemeColor(theme, "secondary")).toBe("#dc004e"); // Should get main color
    });

    it("resolves nested theme color paths", () => {
        expect(resolveThemeColor(theme, "text.primary")).toBe("rgba(0, 0, 0, 0.87)");
        expect(resolveThemeColor(theme, "text.secondary")).toBe("rgba(0, 0, 0, 0.6)");
        expect(resolveThemeColor(theme, "grey.500")).toBe("#9e9e9e");
        expect(resolveThemeColor(theme, "grey.100")).toBe("#f5f5f5");
    });

    it("resolves specific color variants", () => {
        expect(resolveThemeColor(theme, "primary.light")).toBe("#42a5f5");
        expect(resolveThemeColor(theme, "primary.dark")).toBe("#1565c0");
        expect(resolveThemeColor(theme, "secondary.light")).toBe("#f48fb1");
        expect(resolveThemeColor(theme, "secondary.dark")).toBe("#880e4f");
    });

    it("returns original color for non-theme colors", () => {
        expect(resolveThemeColor(theme, "#ff0000")).toBe("#ff0000");
        expect(resolveThemeColor(theme, "red")).toBe("red");
        expect(resolveThemeColor(theme, "rgb(255, 0, 0)")).toBe("rgb(255, 0, 0)");
        expect(resolveThemeColor(theme, "rgba(255, 0, 0, 0.5)")).toBe("rgba(255, 0, 0, 0.5)");
    });

    it("returns original color for invalid theme paths", () => {
        expect(resolveThemeColor(theme, "invalid.color")).toBe("invalid.color");
        expect(resolveThemeColor(theme, "text.invalid")).toBe("text.invalid");
        expect(resolveThemeColor(theme, "primary.invalid")).toBe("primary.invalid");
        expect(resolveThemeColor(theme, "deeply.nested.invalid.path")).toBe("deeply.nested.invalid.path");
    });

    it("handles edge cases with dot notation", () => {
        expect(resolveThemeColor(theme, ".")).toBe(".");
        expect(resolveThemeColor(theme, "..")).toBe("..");
        expect(resolveThemeColor(theme, "primary.")).toBe("primary.");
        expect(resolveThemeColor(theme, ".primary")).toBe(".primary");
    });

    it("handles deep nesting correctly", () => {
        // Test paths that traverse beyond the valid path
        // "primary.main.extra" - finds "primary.main" which is a valid string, but can't go deeper
        // The function will find a valid path partway through and use that
        expect(resolveThemeColor(theme, "primary.main.extra")).toBe("#1976d2"); // Gets primary.main
        
        // "invalid.primary.extra" - no valid path from the start
        expect(resolveThemeColor(theme, "invalid.primary.extra")).toBe("invalid.primary.extra");
    });

    it("handles PaletteColor objects correctly", () => {
        // When accessing a color object directly, should return the main color
        expect(resolveThemeColor(theme, "primary")).toBe("#1976d2");
        expect(resolveThemeColor(theme, "secondary")).toBe("#dc004e");
    });

    it("handles custom theme structure", () => {
        const customTheme = createTheme({
            palette: {
                custom: {
                    brand: "#00ff00",
                    accent: {
                        main: "#ff00ff",
                        light: "#ff99ff",
                    },
                },
            },
        });

        expect(resolveThemeColor(customTheme as any, "custom.brand")).toBe("#00ff00");
        expect(resolveThemeColor(customTheme as any, "custom.accent")).toBe("#ff00ff");
        expect(resolveThemeColor(customTheme as any, "custom.accent.light")).toBe("#ff99ff");
    });
});
