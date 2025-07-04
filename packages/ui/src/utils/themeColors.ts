// AI_CHECK: TYPE_SAFETY=replaced-any-with-unknown-and-type-guards | LAST: 2025-06-28
import type { PaletteColor, Theme } from "@mui/material";

/**
 * Resolves a color value from the MUI theme, supporting dot notation (e.g., "text.secondary", "grey.500").
 * If the color is not found in the theme, returns the original color value.
 * 
 * @param theme - The MUI theme object
 * @param color - The color value to resolve. Can be a theme color path (e.g., "text.secondary") or any valid CSS color
 * @returns The resolved color value
 * 
 * @example
 * ```tsx
 * const theme = useTheme();
 * const color = resolveThemeColor(theme, "text.secondary");
 * // Returns the actual color value from theme.palette.text.secondary
 * ```
 */
export function resolveThemeColor(theme: Theme, color: string | undefined | null): string {
    if (!color) return "currentColor";

    // Handle dot notation (e.g., "text.secondary", "grey.500")
    const colorPath = color.split(".");
    let paletteColor: unknown = theme.palette;

    // Traverse the color path
    for (const key of colorPath) {
        if (paletteColor && typeof paletteColor === "object" && paletteColor !== null) {
            paletteColor = (paletteColor as Record<string, unknown>)[key];
        } else {
            break;
        }
    }

    // If we found a valid color value, use it
    if (typeof paletteColor === "string") {
        return paletteColor;
    }
    
    if (paletteColor && typeof paletteColor === "object" && paletteColor !== null && "main" in paletteColor) {
        const colorObj = paletteColor as PaletteColor;
        return colorObj.main;
    }

    // Return the original color if not found in theme
    return color;
} 
