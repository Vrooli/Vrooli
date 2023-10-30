/** @type {import('ts-jest').JestConfigWithTsJest} */
module.exports = {
    preset: "ts-jest",
    testEnvironment: "node",
    roots: ["src"],
    testPathIgnorePatterns: ["/node_modules/", "/dist/", "/rust/"],
    coveragePathIgnorePatterns: ["/node_modules/", "/dist/", "/rust/"],
};