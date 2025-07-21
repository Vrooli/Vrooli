import { ModelType, DUMMY_ID } from "@vrooli/shared";
import { describe, expect, it, vi, beforeAll } from "vitest";
import { initIdGenerator } from "@vrooli/shared";
import { oneIsPublic } from "./oneIsPublic.js";

describe("oneIsPublic", () => {
    beforeAll(async () => {
        await initIdGenerator();
    });

    describe("basic functionality", () => {
        it("should return true when at least one field is public", () => {
            // Using real models that have isPublic validators
            const list: [string, ModelType][] = [
                ["user", ModelType.User],
                ["project", ModelType.Project],
            ];

            const permissionsData = {
                user: { 
                    id: DUMMY_ID, 
                    isPrivate: false, // User with isPrivate=false is public
                    handle: "testuser",
                },
                project: { 
                    id: DUMMY_ID, 
                    isPrivate: true, // Private project
                    handle: "testproject",
                },
            };

            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(true); // Should be true because user is public
        });

        it("should return false when all fields are private", () => {
            const list: [string, ModelType][] = [
                ["team1", ModelType.Team],
                ["team2", ModelType.Team],
            ];

            const permissionsData = {
                team1: { 
                    id: DUMMY_ID, 
                    isOpenToNewMembers: false, // Private team
                    handle: "team1",
                },
                team2: { 
                    id: DUMMY_ID, 
                    isOpenToNewMembers: false, // Private team
                    handle: "team2",
                },
            };

            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(false);
        });

        it("should handle empty list", () => {
            const list: [string, ModelType][] = [];
            const permissionsData = {};
            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(false);
        });
    });

    describe("null/undefined handling", () => {
        it("should skip null fields", () => {
            const list: [string, ModelType][] = [["routine", ModelType.Routine]];
            const permissionsData = { routine: null, id: DUMMY_ID };
            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(false);
        });

        it("should skip undefined fields", () => {
            const list: [string, ModelType][] = [["api", ModelType.Api]];
            const permissionsData = { id: DUMMY_ID };
            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(false);
        });

        it("should handle falsy but defined values like empty string or 0", () => {
            const list: [string, ModelType][] = [
                ["emptyString", ModelType.Code],
                ["zero", ModelType.Code],
            ];
            const permissionsData = { 
                emptyString: "",
                zero: 0,
                id: DUMMY_ID,
            };
            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(false);
            // Empty string and 0 are falsy, so they won't pass the if check
        });
    });

    describe("multiple fields checking", () => {
        it("should stop checking after finding first public field", () => {
            const list: [string, ModelType][] = [
                ["note", ModelType.Note],
                ["reminder", ModelType.Reminder],
                ["schedule", ModelType.Schedule],
            ];

            const permissionsData = {
                note: { 
                    id: DUMMY_ID,
                    isPrivate: true, // Private
                },
                reminder: { 
                    id: DUMMY_ID,
                    reminderList: {
                        isPrivate: false, // Public via parent
                    },
                },
                schedule: { 
                    id: DUMMY_ID,
                    labels: [], // Would need to check but won't because reminder is public
                },
            };

            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(true);
        });

        it("should check all fields when none are public", () => {
            const list: [string, ModelType][] = [
                ["tag1", ModelType.Tag],
                ["tag2", ModelType.Tag],
                ["tag3", ModelType.Tag],
            ];

            const permissionsData = {
                tag1: { id: DUMMY_ID, tag: "private1" },
                tag2: { id: DUMMY_ID, tag: "private2" },
                tag3: { id: DUMMY_ID, tag: "private3" },
            };

            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(false);
        });
    });

    describe("complex scenarios", () => {
        it("should handle mixed present and missing data", () => {
            const list: [string, ModelType][] = [
                ["present", ModelType.User],
                ["missing", ModelType.Project],
                ["alsoPresent", ModelType.Team],
            ];

            const permissionsData = {
                present: { 
                    id: DUMMY_ID, 
                    isPrivate: false, // Public user
                    handle: "publicuser",
                },
                // missing is undefined
                alsoPresent: { 
                    id: DUMMY_ID, 
                    isOpenToNewMembers: false, // Private team
                    handle: "privateteam",
                },
                id: DUMMY_ID,
            };

            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(true); // True because user is public
        });

        it("should handle real model types with parent relationships", () => {
            const list: [string, ModelType][] = [["meeting", ModelType.Meeting]];
            
            const permissionsData = { 
                meeting: { 
                    id: DUMMY_ID,
                    team: {
                        id: DUMMY_ID,
                        isOpenToNewMembers: true, // Public team makes meeting public
                    },
                }, 
            };
            
            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(true);
        });
    });

    describe("edge cases with real models", () => {
        it("should handle models without explicit isPrivate field", () => {
            const list: [string, ModelType][] = [
                ["award", ModelType.Award],
            ];

            const permissionsData = {
                award: {
                    id: DUMMY_ID,
                    category: "Coding", // Awards are always public
                    progress: 50,
                },
            };

            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(true); // Awards are always public
        });

        it("should handle version models with parent objects", () => {
            const list: [string, ModelType][] = [
                ["resourceVersion", ModelType.ResourceVersion],
            ];

            const permissionsData = {
                resourceVersion: {
                    id: DUMMY_ID,
                    isPrivate: false, // Version itself is public
                    resource: {
                        id: DUMMY_ID,
                    },
                },
            };

            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(true);
        });
    });
});
