module.exports = {
    // Point to your translations directory
    localesPath: "packages/shared/src/translations/locales",
    // Use the root directory as the source path
    srcPath: "packages",
    // Use include patterns to specify which directories to scan
    include: [
        "packages/jobs/src/**/*",
        "packages/server/src/**/*",
        "packages/shared/src/**/*",
        "packages/ui/src/**/*",
    ],
    // Optionally exclude certain patterns
    exclude: [
        "**/node_modules/**",
        "**/dist/**",
        "**/build/**",
        "**/*.test.*",
        "**/*.spec.*",
    ],
    // File extensions to scan
    srcExtensions: ["ts", "tsx", "js", "jsx"],
    // Pattern to match i18next usage
    translationKeyMatcher: [
        // Standard t() function calls
        "/(?:[$ .](t))(['\"`]([^'\"`]+)['\"`].*?)/gi",
        // Variable usage in t()
        "/(?:[$ .](t))(([^)]+))/gi",
        // useTranslation hook
        "/useTranslation(['\"`]([^'\"`]+)['\"`])/gi",
        // Template literals
        "/(?:[$ .](t))(`([^`]+)`)/gi",
    ],
    // Keys that should never be marked as unused
    excludeKey: [
        // Keys that are used dynamically, so they can't be detected
        "_Description_Short",
        "_Name",
        "_Body",
        "_Title",
        "_Label",
        "_Help",
    ],
};
