import { PaletteColor, Theme } from "@mui/material";

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
    let paletteColor: any = theme.palette;

    // Traverse the color path
    for (const key of colorPath) {
        if (paletteColor && typeof paletteColor === "object") {
            paletteColor = paletteColor[key as keyof typeof paletteColor];
        } else {
            break;
        }
    }

    // If we found a valid color value, use it
    if (paletteColor && (typeof paletteColor === "string" || typeof paletteColor === "object")) {
        return typeof paletteColor === "object" && "main" in paletteColor
            ? (paletteColor as PaletteColor).main
            : paletteColor;
    }

    // Return the original color if not found in theme
    return color;
} 
