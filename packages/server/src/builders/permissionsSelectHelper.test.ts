/**
 * NEW VERSION: permissionsSelectHelper.test.ts
 * 
 * Migration from excessive mocking to real dependencies approach.
 * This replaces the old test file with a better testing strategy.
 */
import { expect, describe, it, beforeAll } from "vitest";
import { type ModelType } from "@vrooli/shared";

// Delay imports that use ModelMap  
/* eslint-disable @typescript-eslint/consistent-type-imports */
let permissionsSelectHelper: typeof import("./permissionsSelectHelper.js").permissionsSelectHelper;
type ModelRegistry = import("./permissionsSelectHelper.js").ModelRegistry;
type PermissionsMap = import("../models/types.js").PermissionsMap;
/* eslint-enable @typescript-eslint/consistent-type-imports */

// Simple test configuration factory (inline for this migration)
function createTestModelRegistry(configs: Record<string, any>): ModelRegistry {
    return {
        get: (type: ModelType) => configs[type] || null,
    };
}

// Simple test configuration factory
function createUserConfig(overrides?: any) {
    return {
        idField: "id",
        validate: () => ({
            permissionsSelect: (userId: string | null) => ({
                id: true,
                name: true,
                email: userId === "user123",
                ...overrides,
            }),
        }),
    };
}

function createTeamConfig() {
    return {
        idField: "id",
        validate: () => ({
            permissionsSelect: () => ({
                id: true,
                handle: true,
            }),
        }),
    };
}

describe("permissionsSelectHelper (Migrated)", () => {
    beforeAll(async () => {
        // Import after setup has run to ensure ModelMap is initialized
        const permissionsModule = await import("./permissionsSelectHelper.js");
        permissionsSelectHelper = permissionsModule.permissionsSelectHelper;
    });

    describe("basic functionality", () => {
        it("returns empty object for empty permissions map", () => {
            const map: PermissionsMap<any> = {};
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(map, "user123", 0, [], registry);
            expect(result).toEqual({});
        });

        it("handles simple boolean fields", () => {
            const map: PermissionsMap<any> = {
                id: true,
                name: true,
                email: true,
            };
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(map, "user123", 0, [], registry);
            expect(result).toEqual({
                id: true,
                name: true,
                email: true,
            });
        });

        it("handles function-based permissions map", () => {
            const mapResolver = (userId: string | null) => ({
                id: true,
                isOwner: userId === "user123",
            });
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(mapResolver, "user123", registry);
            expect(result).toEqual({
                id: true,
                isOwner: true,
            });
        });

        it("respects omitFields parameter", () => {
            const map: PermissionsMap<any> = {
                id: true,
                name: true,
                email: true,
                password: true,
            };
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(
                map, 
                "user123", 
                0, 
                ["password", "email"],
                registry,
            );
            expect(result).toEqual({
                id: true,
                name: true,
            });
        });
    });

    describe("nested object handling", () => {
        it("recursively processes nested objects", () => {
            const map: PermissionsMap<any> = {
                id: true,
                profile: {
                    bio: true,
                    avatar: true,
                },
            };
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(map, "user123", 0, [], registry);
            expect(result).toEqual({
                id: true,
                profile: {
                    bio: true,
                    avatar: true,
                },
            });
        });

        it("handles omitFields with dot notation for nested objects", () => {
            const map: PermissionsMap<any> = {
                id: true,
                profile: {
                    bio: true,
                    avatar: true,
                    privateInfo: true,
                },
            };
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(
                map, 
                "user123", 
                0, 
                ["profile.privateInfo"],
                registry,
            );
            expect(result).toEqual({
                id: true,
                profile: {
                    bio: true,
                    avatar: true,
                },
            });
        });

        it("handles deeply nested structures", () => {
            const map: PermissionsMap<any> = {
                id: true,
                organization: {
                    name: true,
                    members: {
                        id: true,
                        role: true,
                        user: {
                            id: true,
                            name: true,
                        },
                    },
                },
            };
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(map, "user123", 0, [], registry);
            expect(result).toEqual({
                id: true,
                organization: {
                    name: true,
                    members: {
                        id: true,
                        role: true,
                        user: {
                            id: true,
                            name: true,
                        },
                    },
                },
            });
        });
    });

    describe("array handling", () => {
        it("processes arrays of permission maps", () => {
            const map: PermissionsMap<any> = {
                id: true,
                tags: [
                    {
                        id: true,
                        name: true,
                    },
                ],
            };
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(map, "user123", 0, [], registry);
            expect(result).toEqual({
                id: true,
                tags: [
                    {
                        id: true,
                        name: true,
                    },
                ],
            });
        });

        it("handles arrays with omitFields", () => {
            const map: PermissionsMap<any> = {
                id: true,
                comments: [
                    {
                        id: true,
                        text: true,
                        private: true,
                    },
                ],
            };
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(
                map, 
                "user123", 
                0, 
                ["comments.private"],
                registry,
            );
            expect(result).toEqual({
                id: true,
                comments: [
                    {
                        id: true,
                        text: true,
                    },
                ],
            });
        });
    });

    describe("ModelType substitution with real configuration", () => {
        it("substitutes ModelType strings with their permission selects", () => {
            const map: PermissionsMap<any> = {
                id: true,
                owner: "User",
            };
            
            // Real configuration data - NOT mocks
            const registry = createTestModelRegistry({
                User: createUserConfig(),
            });
            
            const result = permissionsSelectHelper(map, "user123", 0, [], registry);
            expect(result).toEqual({
                id: true,
                owner: {
                    select: {
                        id: true,
                        name: true,
                        email: true,
                    },
                },
            });
        });

        it("handles ModelType string without validator", () => {
            const map: PermissionsMap<any> = {
                id: true,
                unknownType: "UnknownModel",
            };
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(map, "user123", 0, [], registry);
            expect(result).toEqual({
                id: true,
                unknownType: "UnknownModel",
            });
        });

        it("handles array substitution with [ModelType, omitFields]", () => {
            const map: PermissionsMap<any> = {
                id: true,
                owner: ["User", ["email"]],
            };
            
            const registry = createTestModelRegistry({
                User: createUserConfig(),
            });
            
            const result = permissionsSelectHelper(map, "user123", 0, [], registry);
            expect(result).toEqual({
                id: true,
                owner: {
                    select: {
                        id: true,
                        name: true,
                    },
                },
            });
        });

        it("handles nested ModelType substitutions", () => {
            // Real nested configuration - User references Team
            const userConfig = createUserConfig({ team: "Team" as const });
            const teamConfig = createTeamConfig();
            
            const registry = createTestModelRegistry({
                User: userConfig,
                Team: teamConfig,
            });

            const map: PermissionsMap<any> = {
                id: true,
                owner: "User",
            };
            
            const result = permissionsSelectHelper(map, "user123", 0, [], registry);
            expect(result).toEqual({
                id: true,
                owner: {
                    select: {
                        id: true,
                        name: true,
                        email: true,
                        team: {
                            select: {
                                id: true,
                                handle: true,
                            },
                        },
                    },
                },
            });
        });
    });

    describe("edge cases and error handling", () => {
        it("prevents infinite recursion", () => {
            const map: PermissionsMap<any> = {
                id: true,
                self: {} as any,
            };
            // Create circular reference
            (map.self as any).self = map.self;
            const registry = createTestModelRegistry({});

            expect(() => permissionsSelectHelper(map, "user123", 0, [], registry)).toThrow("0386");
        });

        it("handles null userId correctly", () => {
            const mapResolver = (userId: string | null) => ({
                id: true,
                isPublic: userId === null,
            });
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(mapResolver, null, 0, [], registry);
            expect(result).toEqual({
                id: true,
                isPublic: true,
            });
        });

        it("handles complex mixed structures", () => {
            const map: PermissionsMap<any> = {
                id: true,
                scalar: "value",
                nested: {
                    field: true,
                    array: [{ item: true }],
                },
                modelRef: "User",
                arraySubstitution: ["Team", ["privateField"]],
            };

            const registry = createTestModelRegistry({
                User: createUserConfig(),
                Team: createTeamConfig(),
            });
            
            const result = permissionsSelectHelper(map, "user123", 0, [], registry);
            expect(result).toHaveProperty("id", true);
            expect(result).toHaveProperty("scalar", "value");
            expect(result).toHaveProperty("nested");
            expect(result).toHaveProperty("modelRef");
            expect(result).toHaveProperty("arraySubstitution");
        });

        it("handles empty arrays", () => {
            const map: PermissionsMap<any> = {
                id: true,
                emptyArray: [],
            };
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(map, "user123", 0, [], registry);
            expect(result).toEqual({
                id: true,
                emptyArray: [],
            });
        });

        it("preserves non-object/non-array values", () => {
            const map: PermissionsMap<any> = {
                id: true,
                number: 42,
                string: "test",
                boolean: false,
                nullValue: null,
            };
            const registry = createTestModelRegistry({});
            
            const result = permissionsSelectHelper(map, "user123", 0, [], registry);
            expect(result).toEqual({
                id: true,
                number: 42,
                string: "test",
                boolean: false,
                nullValue: null,
            });
        });
    });
});

describe("Migration Benefits Demonstration", () => {
    it("shows how configuration injection enables flexible testing", () => {
        // Same test logic, different configurations
        const restrictiveConfig = createUserConfig({ name: false }); // Hide name
        const permissiveConfig = createUserConfig({ extra: true }); // Show extra field
        
        const map: PermissionsMap<any> = {
            id: true,
            owner: "User",
        };
        
        const restrictiveRegistry = createTestModelRegistry({
            User: restrictiveConfig,
        });
        
        const permissiveRegistry = createTestModelRegistry({
            User: permissiveConfig,
        });
        
        const restrictiveResult = permissionsSelectHelper(map, "user123", restrictiveRegistry);
        const permissiveResult = permissionsSelectHelper(map, "user123", permissiveRegistry);
        
        // Same function, different behavior based on real configuration
        expect(restrictiveResult.owner.select).toHaveProperty("name", false);
        expect(permissiveResult.owner.select).toHaveProperty("extra", true);
    });
});
