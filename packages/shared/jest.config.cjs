/** @type {import('ts-jest').JestConfigWithTsJest} */
module.exports = {
    roots: ["<rootDir>/src"],
    collectCoverageFrom: [
        "src/**/*.{js,jsx,ts,tsx}",
        "!src/**/*.d.ts",
        "!src/mocks/**",
    ],
    coveragePathIgnorePatterns: ["/node_modules/", "/dist/", "/rust/", "/generated/"],
    setupFilesAfterEnv: ["jest-27-expect-message"],
    extensionsToTreatAsEsm: [".ts"],
    testEnvironment: "jsdom",
    testPathIgnorePatterns: ["/node_modules/", "/dist/", "/rust/", "/generated/"],
    modulePaths: ["<rootDir>/src"],
    transform: {
        "^.+\\.(ts|js|tsx|jsx)$": "<rootDir>/config/jest/swcTransform.cjs",
        "^(?!.*\\.(js|jsx|mjs|cjs|ts|tsx|css|json)$)":
            "<rootDir>/config/jest/fileTransform.cjs",
    },
    moduleFileExtensions: [
        // Place tsx and ts to beginning as suggestion from Jest team
        // https://jestjs.io/docs/configuration#modulefileextensions-arraystring
        "tsx",
        "ts",
        "web.js",
        "js",
        "web.ts",
        "web.tsx",
        "json",
        "web.jsx",
        "jsx",
        "node",
    ],
    resetMocks: true,
};
