// Looking to change the config because the shared package imports aren't working? 
// Instead, try building the shared package and then running the tests again.
module.exports = {
    roots: ["<rootDir>/src"],
    collectCoverageFrom: [
        "src/**/*.{js,jsx,ts,tsx}",
        "!src/**/*.d.ts",
        "!src/mocks/**",
    ],
    coveragePathIgnorePatterns: ["/node_modules/", "/dist/", "/rust/", "/generated/"],
    setupFilesAfterEnv: ["./config/jest/setupTests.ts", "jest-27-expect-message"],
    testEnvironment: "jsdom",
    testPathIgnorePatterns: ["/node_modules/", "/dist/", "/rust/", "/generated/"],
    modulePaths: ["<rootDir>/src"],
    transform: {
        "^.+\\.(ts|js|tsx|jsx)$": "<rootDir>/config/jest/swcTransform.cjs",
        "^.+\\.css$": "<rootDir>/config/jest/cssTransform.cjs",
        "^(?!.*\\.(js|jsx|mjs|cjs|ts|tsx|css|json)$)":
            "<rootDir>/config/jest/fileTransform.cjs",
    },
    transformIgnorePatterns: [
        // "[/\\\\]node_modules[/\\\\].+\\.(js|jsx|mjs|cjs|ts|tsx)$",
        "^.+\\.module\\.(css|sass|scss)$",
    ],
    moduleNameMapper: {
        "^react-native$": "react-native-web",
        "^.+\\.module\\.(css|sass|scss)$": "identity-obj-proxy",
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
