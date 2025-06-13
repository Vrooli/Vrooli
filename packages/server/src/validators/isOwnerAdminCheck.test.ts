import { describe, expect, it } from "vitest";
import { isOwnerAdminCheck } from "./isOwnerAdminCheck.js";

describe("isOwnerAdminCheck", () => {
    describe("user authentication", () => {
        it("should return false when userId is null", () => {
            const owner = { User: { id: "user123" } };
            expect(isOwnerAdminCheck(owner, null)).toBe(false);
        });

        it("should return false when userId is undefined", () => {
            const owner = { User: { id: "user123" } };
            expect(isOwnerAdminCheck(owner, undefined)).toBe(false);
        });
    });

    describe("user ownership", () => {
        it("should return true when user owns the object (string comparison)", () => {
            const owner = { User: { id: "user123" } };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(true);
        });

        it("should return false when user does not own the object", () => {
            const owner = { User: { id: "user123" } };
            expect(isOwnerAdminCheck(owner, "user456")).toBe(false);
        });

        it("should handle numeric IDs converted to string", () => {
            const owner = { User: { id: 123 } };
            expect(isOwnerAdminCheck(owner, "123")).toBe(true);
        });

        it("should handle numeric userId compared with string ID", () => {
            const owner = { User: { id: "123" } };
            expect(isOwnerAdminCheck(owner, "123")).toBe(true);
        });

        it("should return false for case-sensitive string comparison", () => {
            const owner = { User: { id: "User123" } };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should handle empty string IDs", () => {
            const owner = { User: { id: "" } };
            expect(isOwnerAdminCheck(owner, "")).toBe(true);
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should handle whitespace in IDs", () => {
            const owner = { User: { id: " user123 " } };
            expect(isOwnerAdminCheck(owner, " user123 ")).toBe(true);
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });
    });

    describe("team ownership", () => {
        it("should return true when user is an admin member of the team", () => {
            const owner = {
                Team: {
                    members: [
                        { userId: "user123", isAdmin: true },
                        { userId: "user456", isAdmin: false },
                    ],
                },
            };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(true);
        });

        it("should return false when user is a non-admin member of the team", () => {
            const owner = {
                Team: {
                    members: [
                        { userId: "user123", isAdmin: false },
                        { userId: "user456", isAdmin: true },
                    ],
                },
            };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should return false when user is not a member of the team", () => {
            const owner = {
                Team: {
                    members: [
                        { userId: "user123", isAdmin: true },
                        { userId: "user456", isAdmin: false },
                    ],
                },
            };
            expect(isOwnerAdminCheck(owner, "user789")).toBe(false);
        });

        it("should handle numeric userIds in team members", () => {
            const owner = {
                Team: {
                    members: [
                        { userId: 123, isAdmin: true },
                        { userId: 456, isAdmin: false },
                    ],
                },
            };
            expect(isOwnerAdminCheck(owner, "123")).toBe(true);
            expect(isOwnerAdminCheck(owner, "456")).toBe(false);
        });

        it("should handle empty team members array", () => {
            const owner = {
                Team: {
                    members: [],
                },
            };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should handle null members array", () => {
            const owner = {
                Team: {
                    members: null,
                },
            };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should handle undefined members array", () => {
            const owner = {
                Team: {
                    members: undefined,
                },
            };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should handle missing members property", () => {
            const owner = {
                Team: {},
            };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should find admin among multiple members", () => {
            const owner = {
                Team: {
                    members: [
                        { userId: "user1", isAdmin: false },
                        { userId: "user2", isAdmin: false },
                        { userId: "user3", isAdmin: true },
                        { userId: "user4", isAdmin: false },
                    ],
                },
            };
            expect(isOwnerAdminCheck(owner, "user3")).toBe(true);
            expect(isOwnerAdminCheck(owner, "user1")).toBe(false);
            expect(isOwnerAdminCheck(owner, "user4")).toBe(false);
        });

        it("should handle members with additional properties", () => {
            const owner = {
                Team: {
                    members: [
                        {
                            userId: "user123",
                            isAdmin: true,
                            name: "John Doe",
                            role: "developer",
                            joinedAt: "2023-01-01",
                        },
                    ],
                },
            };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(true);
        });
    });

    describe("neither user nor team", () => {
        it("should return false when owner has neither User nor Team", () => {
            const owner = {};
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should return false when owner has other properties but not User or Team", () => {
            const owner = {
                Organization: { id: "org123" },
                SomeOtherEntity: { id: "entity123" },
            };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should return false when User property is null", () => {
            const owner = { User: null };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should return false when Team property is null", () => {
            const owner = { Team: null };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should return false when User property is undefined", () => {
            const owner = { User: undefined };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should return false when Team property is undefined", () => {
            const owner = { Team: undefined };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });
    });

    describe("precedence when both User and Team exist", () => {
        it("should check User first when both User and Team exist", () => {
            const owner = {
                User: { id: "user123" },
                Team: {
                    members: [
                        { userId: "user123", isAdmin: false },
                    ],
                },
            };
            // Should return true because User check succeeds first
            expect(isOwnerAdminCheck(owner, "user123")).toBe(true);
        });

        it("should not fall through to Team when User exists (User takes precedence)", () => {
            const owner = {
                User: { id: "user456" },
                Team: {
                    members: [
                        { userId: "user123", isAdmin: true },
                    ],
                },
            };
            // Should return false because User check fails and Team is not checked when User exists
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should return false when both User and Team checks fail", () => {
            const owner = {
                User: { id: "user456" },
                Team: {
                    members: [
                        { userId: "user789", isAdmin: true },
                    ],
                },
            };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });
    });

    describe("edge cases and robustness", () => {
        it("should handle malformed User object without id", () => {
            const owner = { User: {} };
            expect(() => isOwnerAdminCheck(owner, "user123")).toThrow();
        });

        it("should throw when User has null id", () => {
            const owner = { User: { id: null } };
            expect(() => isOwnerAdminCheck(owner, "user123")).toThrow();
        });

        it("should throw when User has undefined id", () => {
            const owner = { User: { id: undefined } };
            expect(() => isOwnerAdminCheck(owner, "user123")).toThrow();
        });

        it("should handle team member without isAdmin property", () => {
            const owner = {
                Team: {
                    members: [
                        { userId: "user123" }, // Missing isAdmin
                    ],
                },
            };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(false);
        });

        it("should handle team member with falsy isAdmin values", () => {
            const owner = {
                Team: {
                    members: [
                        { userId: "user1", isAdmin: false },
                        { userId: "user2", isAdmin: 0 },
                        { userId: "user3", isAdmin: null },
                        { userId: "user4", isAdmin: undefined },
                        { userId: "user5", isAdmin: "" },
                    ],
                },
            };
            expect(isOwnerAdminCheck(owner, "user1")).toBe(false);
            expect(isOwnerAdminCheck(owner, "user2")).toBe(false);
            expect(isOwnerAdminCheck(owner, "user3")).toBe(false);
            expect(isOwnerAdminCheck(owner, "user4")).toBe(false);
            expect(isOwnerAdminCheck(owner, "user5")).toBe(false);
        });

        it("should handle team member with truthy isAdmin values", () => {
            const owner = {
                Team: {
                    members: [
                        { userId: "user1", isAdmin: true },
                        { userId: "user2", isAdmin: 1 },
                        { userId: "user3", isAdmin: "true" },
                        { userId: "user4", isAdmin: {} },
                    ],
                },
            };
            expect(isOwnerAdminCheck(owner, "user1")).toBe(true);
            expect(isOwnerAdminCheck(owner, "user2")).toBe(true);
            expect(isOwnerAdminCheck(owner, "user3")).toBe(true);
            expect(isOwnerAdminCheck(owner, "user4")).toBe(true);
        });

        it("should handle very large team member arrays", () => {
            const members = [];
            for (let i = 0; i < 1000; i++) {
                members.push({ userId: `user${i}`, isAdmin: i === 500 });
            }
            const owner = { Team: { members } };
            
            expect(isOwnerAdminCheck(owner, "user500")).toBe(true);
            expect(isOwnerAdminCheck(owner, "user499")).toBe(false);
            expect(isOwnerAdminCheck(owner, "user501")).toBe(false);
        });

        it("should handle complex nested owner object", () => {
            const owner = {
                User: {
                    id: "user123",
                    name: "John Doe",
                    profile: {
                        avatar: "avatar.jpg",
                        settings: {
                            theme: "dark",
                        },
                    },
                },
                otherProperty: "should be ignored",
            };
            expect(isOwnerAdminCheck(owner, "user123")).toBe(true);
        });
    });
});
