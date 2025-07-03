import { ModelType } from "@vrooli/shared";
import { describe, expect, it, vi } from "vitest";
import { ModelMap } from "../models/base/index.js";
import { oneIsPublic } from "./oneIsPublic.js";

// Mock ModelMap
vi.mock("../models/base/index.js", () => ({
    ModelMap: {
        getLogic: vi.fn(),
    },
}));

describe("oneIsPublic", () => {
    // Helper to create a mock validator
    const createMockValidator = (isPublicResult: boolean) => ({
        isPublic: vi.fn().mockReturnValue(isPublicResult),
    });

    // Helper to setup ModelMap mock
    const setupModelMapMock = (modelType: ModelType, validator: any, idField = "id") => {
        (ModelMap.getLogic as any).mockImplementation((fields: string[], type: string) => {
            if (type === modelType) {
                return {
                    idField,
                    validate: () => validator,
                };
            }
            throw new Error(`Unexpected model type: ${type}`);
        });
    };

    describe("basic functionality", () => {
        it("should return true when at least one field is public", () => {
            const publicValidator = createMockValidator(true);
            const privateValidator = createMockValidator(false);

            // Setup different validators for different types
            (ModelMap.getLogic as any).mockImplementation((fields: string[], type: string) => {
                if (type === ModelType.User) {
                    return { idField: "id", validate: () => publicValidator };
                } else if (type === ModelType.Project) {
                    return { idField: "id", validate: () => privateValidator };
                }
            });

            const list: [string, ModelType][] = [
                ["user", ModelType.User],
                ["project", ModelType.Project],
            ];

            const permissionsData = {
                user: { id: "user123", name: "Test User" },
                project: { id: "project456", title: "Test Project" },
            };

            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(true);
            expect(publicValidator.isPublic).toHaveBeenCalledWith(permissionsData.user, getParentInfo);
            // Since the first field returns true, it short-circuits and doesn't check the second
            expect(privateValidator.isPublic).not.toHaveBeenCalled();
        });

        it("should return false when all fields are private", () => {
            const privateValidator = createMockValidator(false);
            setupModelMapMock(ModelType.Team, privateValidator);

            const list: [string, ModelType][] = [
                ["team1", ModelType.Team],
                ["team2", ModelType.Team],
            ];

            const permissionsData = {
                team1: { id: "team1", name: "Team 1" },
                team2: { id: "team2", name: "Team 2" },
            };

            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(false);
            expect(privateValidator.isPublic).toHaveBeenCalledTimes(2);
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
        it("should skip null fields and not call validator", () => {
            const validator = createMockValidator(true);
            setupModelMapMock(ModelType.Routine, validator);

            const list: [string, ModelType][] = [["routine", ModelType.Routine]];
            const permissionsData = { routine: null, id: "parent123" };
            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData as any, getParentInfo);
            expect(result).toBe(false);
            // Due to the if check, validator should not be called for null fields
            expect(validator.isPublic).not.toHaveBeenCalled();
            expect(getParentInfo).not.toHaveBeenCalled();
        });

        it("should skip undefined fields and not call validator", () => {
            const validator = createMockValidator(false);
            setupModelMapMock(ModelType.Api, validator);

            const list: [string, ModelType][] = [["api", ModelType.Api]];
            const permissionsData = { id: "main123" };
            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(false);
            // Due to the if check, validator should not be called for undefined fields
            expect(validator.isPublic).not.toHaveBeenCalled();
            expect(getParentInfo).not.toHaveBeenCalled();
        });

        it("should handle falsy but defined values like empty string or 0", () => {
            const validator = createMockValidator(true);
            setupModelMapMock(ModelType.Code, validator);

            const list: [string, ModelType][] = [
                ["emptyString", ModelType.Code],
                ["zero", ModelType.Code],
            ];
            const permissionsData = { 
                emptyString: "",
                zero: 0,
                id: "test123",
            };
            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData as any, getParentInfo);
            expect(result).toBe(false);
            // Empty string and 0 are falsy, so they won't pass the if check
            expect(validator.isPublic).not.toHaveBeenCalled();
        });
    });

    describe("custom idField handling", () => {
        it("should work with valid data and custom idField", () => {
            const validator = createMockValidator(true);
            const customIdField = "customId";
            
            (ModelMap.getLogic as any).mockImplementation(() => ({
                idField: customIdField,
                validate: () => validator,
            }));

            const list: [string, ModelType][] = [["standard", ModelType.Standard]];
            const permissionsData = { 
                standard: { customId: "std123", data: "some data" },
                customId: "custom123",
            };
            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData as any, getParentInfo);
            expect(result).toBe(true);
            expect(validator.isPublic).toHaveBeenCalledWith(permissionsData.standard, getParentInfo);
        });

        it("should handle objects with custom idField", () => {
            const validator = createMockValidator(false);
            
            (ModelMap.getLogic as any).mockImplementation(() => ({
                idField: "specificId",
                validate: () => validator,
            }));

            const list: [string, ModelType][] = [["resource", ModelType.Resource]];
            const permissionsData = { 
                resource: { specificId: "res123", name: "Resource" },
                id: "fallback123",
            };
            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData as any, getParentInfo);
            expect(result).toBe(false);
            expect(validator.isPublic).toHaveBeenCalledWith(permissionsData.resource, getParentInfo);
        });
    });

    describe("multiple fields checking", () => {
        it("should stop checking after finding first public field", () => {
            const publicValidator = createMockValidator(true);
            const privateValidator = createMockValidator(false);

            (ModelMap.getLogic as any).mockImplementation((fields: string[], type: string) => {
                if (type === ModelType.Note) {
                    return { idField: "id", validate: () => privateValidator };
                } else if (type === ModelType.Reminder) {
                    return { idField: "id", validate: () => publicValidator };
                } else if (type === ModelType.Schedule) {
                    return { idField: "id", validate: () => privateValidator };
                }
            });

            const list: [string, ModelType][] = [
                ["note", ModelType.Note],
                ["reminder", ModelType.Reminder],
                ["schedule", ModelType.Schedule],
            ];

            const permissionsData = {
                note: { id: "note1" },
                reminder: { id: "reminder1" },
                schedule: { id: "schedule1" },
            };

            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(true);
            
            // Should check first two, but not the third since second returned true
            expect(privateValidator.isPublic).toHaveBeenCalledTimes(1);
            expect(publicValidator.isPublic).toHaveBeenCalledTimes(1);
        });

        it("should check all fields when none are public", () => {
            const privateValidator = createMockValidator(false);
            setupModelMapMock(ModelType.Tag, privateValidator);

            const list: [string, ModelType][] = [
                ["tag1", ModelType.Tag],
                ["tag2", ModelType.Tag],
                ["tag3", ModelType.Tag],
            ];

            const permissionsData = {
                tag1: { id: "t1" },
                tag2: { id: "t2" },
                tag3: { id: "t3" },
            };

            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(false);
            expect(privateValidator.isPublic).toHaveBeenCalledTimes(3);
        });
    });

    describe("complex scenarios", () => {
        it("should handle mixed present and missing data", () => {
            const publicValidator = createMockValidator(true);
            const privateValidator = createMockValidator(false);
            
            (ModelMap.getLogic as any).mockImplementation((fields: string[], type: string) => ({
                idField: "id",
                validate: () => type === ModelType.User ? publicValidator : privateValidator,
            }));

            const list: [string, ModelType][] = [
                ["present", ModelType.User],
                ["missing", ModelType.Project],
                ["alsoPresent", ModelType.Team],
            ];

            const permissionsData = {
                present: { id: "p1", data: "some data" },
                // missing is undefined
                alsoPresent: { id: "p2", data: "more data" },
                id: "mainId",
            };

            const getParentInfo = vi.fn();

            const result = oneIsPublic(list, permissionsData, getParentInfo);
            expect(result).toBe(true);
            
            // Only the first field should be checked since it returns true
            expect(publicValidator.isPublic).toHaveBeenCalledWith(permissionsData.present, getParentInfo);
            // Missing field should be skipped (undefined)
            expect(privateValidator.isPublic).not.toHaveBeenCalled();
            expect(getParentInfo).not.toHaveBeenCalled();
        });

        it("should handle validator throwing errors", () => {
            const errorValidator = {
                isPublic: vi.fn().mockImplementation(() => {
                    throw new Error("Validator error");
                }),
            };
            
            setupModelMapMock(ModelType.Label, errorValidator);

            const list: [string, ModelType][] = [["label", ModelType.Label]];
            const permissionsData = { label: { id: "label1" } };
            const getParentInfo = vi.fn();

            expect(() => oneIsPublic(list, permissionsData, getParentInfo)).toThrow("Validator error");
        });

        it("should pass getParentInfo through to validators correctly", () => {
            const validator = createMockValidator(false);
            setupModelMapMock(ModelType.Meeting, validator);

            const list: [string, ModelType][] = [["meeting", ModelType.Meeting]];
            const permissionsData = { meeting: { id: "m1" } };
            const getParentInfo = vi.fn();

            oneIsPublic(list, permissionsData, getParentInfo);
            
            // Verify getParentInfo is passed as second argument to validator
            expect(validator.isPublic).toHaveBeenCalledWith(permissionsData.meeting, getParentInfo);
        });
    });
});
