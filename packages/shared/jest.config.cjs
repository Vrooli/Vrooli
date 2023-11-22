/** @type {import('ts-jest').JestConfigWithTsJest} */
module.exports = {
    preset: "ts-jest",
    testEnvironment: "node",
    roots: ["<rootDir>/src"],
    testPathIgnorePatterns: ["/node_modules/", "/dist/"],
    coveragePathIgnorePatterns: ["/node_modules/", "/dist/"],
};