// DOC: docs/internal/UTILS_UNIFICATION_NOTES.md
/**
 * Convert a display label into a URL/testId-safe slug.
 *
 * Lowercases, replaces whitespace and special characters with hyphens,
 * and strips anything not alphanumeric or hyphen.
 *
 * Used by TerminalLauncher and MobileToolbar to generate consistent
 * data-testid attributes from dynamic labels.
 */
export function slugify(label) {
    return label
        .toLowerCase()
        .replace(/[+\s]+/g, "-")
        .replace(/[^\w-]/g, "");
}
