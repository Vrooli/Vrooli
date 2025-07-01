// AI_CHECK: TEST_COVERAGE=1 | TEST_QUALITY=1 | LAST: 2025-06-24
import { describe, it, expect } from "vitest";
import { createTailwindThemeClasses, cn } from "./tailwind-theme.js";
import { createTheme } from "@mui/material/styles";

describe("createTailwindThemeClasses", () => {
    const mockTheme = createTheme();

    it("should return an object with all expected properties", () => {
        const classes = createTailwindThemeClasses(mockTheme);

        // Verify structure exists
        expect(classes).toHaveProperty("bgDefault");
        expect(classes).toHaveProperty("bgPaper");
        expect(classes).toHaveProperty("textPrimary");
        expect(classes).toHaveProperty("textSecondary");
        expect(classes).toHaveProperty("buttonPrimary");
        expect(classes).toHaveProperty("buttonSecondary");
        expect(classes).toHaveProperty("cardContainer");
        expect(classes).toHaveProperty("flexCenter");
        expect(classes).toHaveProperty("spacing");
        expect(classes).toHaveProperty("margin");
        expect(classes).toHaveProperty("padding");
    });

    it("should return consistent string values for all class properties", () => {
        const classes = createTailwindThemeClasses(mockTheme);

        // All properties should return strings
        expect(typeof classes.bgDefault).toBe("string");
        expect(typeof classes.bgPaper).toBe("string");
        expect(typeof classes.textPrimary).toBe("string");
        expect(typeof classes.textSecondary).toBe("string");
        expect(typeof classes.buttonPrimary).toBe("string");
        expect(typeof classes.buttonSecondary).toBe("string");
        expect(typeof classes.cardContainer).toBe("string");
        expect(typeof classes.flexCenter).toBe("string");

        // All strings should be non-empty
        expect(classes.bgDefault.length).toBeGreaterThan(0);
        expect(classes.textPrimary.length).toBeGreaterThan(0);
        expect(classes.flexCenter.length).toBeGreaterThan(0);
    });

    it("should provide spacing utility functions that return strings", () => {
        const classes = createTailwindThemeClasses(mockTheme);

        expect(typeof classes.spacing).toBe("function");
        expect(typeof classes.margin).toBe("function");
        expect(typeof classes.padding).toBe("function");

        // Functions should return strings
        expect(typeof classes.spacing(4)).toBe("string");
        expect(typeof classes.margin(2)).toBe("string");
        expect(typeof classes.padding(6)).toBe("string");

        // Return values should be non-empty
        expect(classes.spacing(4).length).toBeGreaterThan(0);
        expect(classes.margin(2).length).toBeGreaterThan(0);
        expect(classes.padding(6).length).toBeGreaterThan(0);
    });

    it("should handle spacing utility with different numeric values", () => {
        const classes = createTailwindThemeClasses(mockTheme);

        // Should handle various numeric inputs
        expect(classes.spacing(0)).toBeDefined();
        expect(classes.spacing(8)).toBeDefined();
        expect(classes.spacing(12)).toBeDefined();
        expect(classes.margin(1)).toBeDefined();
        expect(classes.margin(10)).toBeDefined();
        expect(classes.padding(3)).toBeDefined();
        expect(classes.padding(5)).toBeDefined();

        // All should return strings
        expect(typeof classes.spacing(0)).toBe("string");
        expect(typeof classes.margin(1)).toBe("string");
        expect(typeof classes.padding(3)).toBe("string");
    });

    it("should work consistently with different theme configurations", () => {
        const customTheme = createTheme({
            palette: {
                primary: { main: "#ff0000" },
                secondary: { main: "#00ff00" },
            },
        });

        const classes = createTailwindThemeClasses(customTheme);

        // Should still return the same structure regardless of theme values
        expect(classes).toHaveProperty("bgDefault");
        expect(classes).toHaveProperty("textPrimary");
        expect(classes).toHaveProperty("flexCenter");
        
        expect(typeof classes.bgDefault).toBe("string");
        expect(typeof classes.textPrimary).toBe("string");
        expect(typeof classes.flexCenter).toBe("string");
    });

    it("should return deterministic results for the same theme", () => {
        const theme = createTheme();
        const classes1 = createTailwindThemeClasses(theme);
        const classes2 = createTailwindThemeClasses(theme);

        // Should return identical results for same theme
        expect(classes1.bgDefault).toBe(classes2.bgDefault);
        expect(classes1.textPrimary).toBe(classes2.textPrimary);
        expect(classes1.flexCenter).toBe(classes2.flexCenter);
        expect(classes1.spacing(4)).toBe(classes2.spacing(4));
    });
});

describe("cn utility function", () => {
    it("should combine multiple class names", () => {
        const result = cn("class1", "class2", "class3");
        expect(result).toBe("class1 class2 class3");
    });

    it("should filter out undefined values", () => {
        const result = cn("class1", undefined, "class2", undefined, "class3");
        expect(result).toBe("class1 class2 class3");
    });

    it("should filter out false values", () => {
        const result = cn("class1", false, "class2", false, "class3");
        expect(result).toBe("class1 class2 class3");
    });

    it("should handle mixed valid and invalid values", () => {
        const result = cn("class1", undefined, false, "class2", "", "class3");
        expect(result).toBe("class1 class2 class3");
    });

    it("should handle empty input", () => {
        const result = cn();
        expect(result).toBe("");
    });

    it("should handle all undefined/false values", () => {
        const result = cn(undefined, false, undefined, false);
        expect(result).toBe("");
    });

    it("should handle single class name", () => {
        const result = cn("single-class");
        expect(result).toBe("single-class");
    });

    it("should handle conditional classes", () => {
        const isActive = true;
        const isDisabled = false;

        const result = cn(
            "base-class",
            isActive && "active-class",
            isDisabled && "disabled-class",
            "always-present",
        );

        expect(result).toBe("base-class active-class always-present");
    });

    it("should work with complex conditional logic", () => {
        const variant = "primary";
        const size = "large";
        const disabled = false;

        const result = cn(
            "btn",
            variant === "primary" && "btn-primary",
            variant === "secondary" && "btn-secondary",
            size === "large" && "btn-lg",
            size === "small" && "btn-sm",
            disabled && "btn-disabled",
        );

        expect(result).toBe("btn btn-primary btn-lg");
    });

    it("should preserve spaces in individual class strings", () => {
        const result = cn("class1 class2", "class3 class4");
        expect(result).toBe("class1 class2 class3 class4");
    });

    it("should maintain order of classes", () => {
        const result = cn("first", "second", "third");
        expect(result).toBe("first second third");
        
        const reverseResult = cn("third", "second", "first");
        expect(reverseResult).toBe("third second first");
    });

    it("should handle string with multiple spaces", () => {
        const result = cn("class1  class2", "class3   class4");
        expect(result).toBe("class1  class2 class3   class4");
    });
});

describe("Integration behavior", () => {
    it("should work together to create dynamic class combinations", () => {
        const theme = createTheme();
        const classes = createTailwindThemeClasses(theme);

        const isActive = true;
        const isPrimary = true;

        const buttonClasses = cn(
            classes.buttonPrimary,
            isActive && "active-state",
            isPrimary && classes.shadowMd,
            classes.borderRadius,
        );

        // Should be a non-empty string containing all expected parts
        expect(typeof buttonClasses).toBe("string");
        expect(buttonClasses.length).toBeGreaterThan(0);
        expect(buttonClasses).toContain("active-state");
    });

    it("should handle spacing utilities with cn", () => {
        const theme = createTheme();
        const classes = createTailwindThemeClasses(theme);

        const containerClasses = cn(
            classes.cardContainer,
            classes.spacing(6),
            classes.flexColumn,
        );

        expect(typeof containerClasses).toBe("string");
        expect(containerClasses.length).toBeGreaterThan(0);
        
        // Should contain all the individual class strings
        expect(containerClasses.split(" ").length).toBeGreaterThanOrEqual(3);
    });

    it("should support responsive and state-based styling patterns", () => {
        const theme = createTheme();
        const classes = createTailwindThemeClasses(theme);

        const isLoading = false;
        const isMobile = true;

        const componentClasses = cn(
            classes.flexCenter,
            classes.textPrimary,
            isMobile && "mobile-styles",
            !isMobile && "desktop-styles",
            isLoading && "loading-state",
            !isLoading && "loaded-state",
        );

        expect(typeof componentClasses).toBe("string");
        expect(componentClasses).toContain("mobile-styles");
        expect(componentClasses).toContain("loaded-state");
        expect(componentClasses).not.toContain("desktop-styles");
        expect(componentClasses).not.toContain("loading-state");
    });

    it("should produce deterministic results for same inputs", () => {
        const theme = createTheme();
        const classes = createTailwindThemeClasses(theme);

        const result1 = cn(classes.buttonPrimary, classes.flexCenter, "custom");
        const result2 = cn(classes.buttonPrimary, classes.flexCenter, "custom");

        expect(result1).toBe(result2);
    });
});
