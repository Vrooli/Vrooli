import { describe, expect, it } from "vitest";
import { defaultPermissions } from "./defaultPermissions.js";

describe("defaultPermissions", () => {
    // Helper function to generate all permission combinations
    const generatePermissionStates = () => {
        const states: Array<{ isAdmin: boolean, isDeleted: boolean, isLoggedIn: boolean, isPublic: boolean }> = [];
        for (const isAdmin of [true, false]) {
            for (const isDeleted of [true, false]) {
                for (const isLoggedIn of [true, false]) {
                    for (const isPublic of [true, false]) {
                        states.push({ isAdmin, isDeleted, isLoggedIn, isPublic });
                    }
                }
            }
        }
        return states;
    };

    describe("canBookmark", () => {
        it("should allow bookmarking when logged in, not deleted, and either admin or public", () => {
            // Allowed scenarios
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: true, isPublic: false }).canBookmark()).toBe(true);
            expect(defaultPermissions({ isAdmin: false, isDeleted: false, isLoggedIn: true, isPublic: true }).canBookmark()).toBe(true);
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: true, isPublic: true }).canBookmark()).toBe(true);
        });

        it("should not allow bookmarking when not logged in", () => {
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: false, isPublic: true }).canBookmark()).toBe(false);
            expect(defaultPermissions({ isAdmin: false, isDeleted: false, isLoggedIn: false, isPublic: true }).canBookmark()).toBe(false);
        });

        it("should not allow bookmarking when deleted", () => {
            expect(defaultPermissions({ isAdmin: true, isDeleted: true, isLoggedIn: true, isPublic: true }).canBookmark()).toBe(false);
        });

        it("should not allow bookmarking when not admin and not public", () => {
            expect(defaultPermissions({ isAdmin: false, isDeleted: false, isLoggedIn: true, isPublic: false }).canBookmark()).toBe(false);
        });
    });

    describe("canComment", () => {
        it("should have the same logic as canBookmark", () => {
            const states = generatePermissionStates();
            for (const state of states) {
                const perms = defaultPermissions(state);
                expect(perms.canComment()).toBe(perms.canBookmark());
            }
        });
    });

    describe("canConnect", () => {
        it("should have the same logic as canBookmark", () => {
            const states = generatePermissionStates();
            for (const state of states) {
                const perms = defaultPermissions(state);
                expect(perms.canConnect()).toBe(perms.canBookmark());
            }
        });
    });

    describe("canCopy", () => {
        it("should have the same logic as canBookmark", () => {
            const states = generatePermissionStates();
            for (const state of states) {
                const perms = defaultPermissions(state);
                expect(perms.canCopy()).toBe(perms.canBookmark());
            }
        });
    });

    describe("canDelete", () => {
        it("should only allow deletion when logged in, not deleted, and admin", () => {
            // Only allowed scenario
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: true, isPublic: true }).canDelete()).toBe(true);
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: true, isPublic: false }).canDelete()).toBe(true);

            // Disallowed scenarios
            expect(defaultPermissions({ isAdmin: false, isDeleted: false, isLoggedIn: true, isPublic: true }).canDelete()).toBe(false);
            expect(defaultPermissions({ isAdmin: true, isDeleted: true, isLoggedIn: true, isPublic: true }).canDelete()).toBe(false);
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: false, isPublic: true }).canDelete()).toBe(false);
        });
    });

    describe("canDisconnect", () => {
        it("should only require being logged in", () => {
            // Allowed scenarios - only needs isLoggedIn
            expect(defaultPermissions({ isAdmin: false, isDeleted: true, isLoggedIn: true, isPublic: false }).canDisconnect()).toBe(true);
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: true, isPublic: true }).canDisconnect()).toBe(true);

            // Disallowed scenarios - not logged in
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: false, isPublic: true }).canDisconnect()).toBe(false);
            expect(defaultPermissions({ isAdmin: false, isDeleted: true, isLoggedIn: false, isPublic: false }).canDisconnect()).toBe(false);
        });
    });

    describe("canRead", () => {
        it("should allow reading when not deleted and either public or admin", () => {
            // Allowed scenarios
            expect(defaultPermissions({ isAdmin: false, isDeleted: false, isLoggedIn: false, isPublic: true }).canRead()).toBe(true);
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: false, isPublic: false }).canRead()).toBe(true);
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: true, isPublic: true }).canRead()).toBe(true);

            // Disallowed scenarios
            expect(defaultPermissions({ isAdmin: false, isDeleted: true, isLoggedIn: true, isPublic: true }).canRead()).toBe(false);
            expect(defaultPermissions({ isAdmin: false, isDeleted: false, isLoggedIn: false, isPublic: false }).canRead()).toBe(false);
        });

        it("should not require login for reading", () => {
            expect(defaultPermissions({ isAdmin: false, isDeleted: false, isLoggedIn: false, isPublic: true }).canRead()).toBe(true);
        });
    });

    describe("canReport", () => {
        it("should allow reporting when logged in, not admin, not deleted, and public", () => {
            // Only allowed scenario
            expect(defaultPermissions({ isAdmin: false, isDeleted: false, isLoggedIn: true, isPublic: true }).canReport()).toBe(true);

            // Disallowed scenarios
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: true, isPublic: true }).canReport()).toBe(false); // Admin can't report
            expect(defaultPermissions({ isAdmin: false, isDeleted: true, isLoggedIn: true, isPublic: true }).canReport()).toBe(false); // Deleted
            expect(defaultPermissions({ isAdmin: false, isDeleted: false, isLoggedIn: false, isPublic: true }).canReport()).toBe(false); // Not logged in
            expect(defaultPermissions({ isAdmin: false, isDeleted: false, isLoggedIn: true, isPublic: false }).canReport()).toBe(false); // Not public
        });
    });

    describe("canRun", () => {
        it("should have the same logic as canRead", () => {
            const states = generatePermissionStates();
            for (const state of states) {
                const perms = defaultPermissions(state);
                expect(perms.canRun()).toBe(perms.canRead());
            }
        });
    });

    describe("canShare", () => {
        it("should have the same logic as canRead", () => {
            const states = generatePermissionStates();
            for (const state of states) {
                const perms = defaultPermissions(state);
                expect(perms.canShare()).toBe(perms.canRead());
            }
        });
    });

    describe("canTransfer", () => {
        it("should allow transfer when logged in, admin, and not deleted", () => {
            // Only allowed scenario
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: true, isPublic: true }).canTransfer()).toBe(true);
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: true, isPublic: false }).canTransfer()).toBe(true);

            // Disallowed scenarios
            expect(defaultPermissions({ isAdmin: false, isDeleted: false, isLoggedIn: true, isPublic: true }).canTransfer()).toBe(false);
            expect(defaultPermissions({ isAdmin: true, isDeleted: true, isLoggedIn: true, isPublic: true }).canTransfer()).toBe(false);
            expect(defaultPermissions({ isAdmin: true, isDeleted: false, isLoggedIn: false, isPublic: true }).canTransfer()).toBe(false);
        });
    });

    describe("canUpdate", () => {
        it("should have the same logic as canDelete", () => {
            const states = generatePermissionStates();
            for (const state of states) {
                const perms = defaultPermissions(state);
                expect(perms.canUpdate()).toBe(perms.canDelete());
            }
        });
    });

    describe("canUse", () => {
        it("should have the same logic as canBookmark", () => {
            const states = generatePermissionStates();
            for (const state of states) {
                const perms = defaultPermissions(state);
                expect(perms.canUse()).toBe(perms.canBookmark());
            }
        });
    });

    describe("canReact", () => {
        it("should have the same logic as canBookmark", () => {
            const states = generatePermissionStates();
            for (const state of states) {
                const perms = defaultPermissions(state);
                expect(perms.canReact()).toBe(perms.canBookmark());
            }
        });
    });

    describe("comprehensive permission matrix", () => {
        it("should return correct permissions for all state combinations", () => {
            // Test a comprehensive matrix of different states
            const testCases = [
                // Admin user scenarios
                { state: { isAdmin: true, isDeleted: false, isLoggedIn: true, isPublic: true }, expectedPermissions: {
                    canBookmark: true, canComment: true, canConnect: true, canCopy: true,
                    canDelete: true, canDisconnect: true, canRead: true, canReport: false,
                    canRun: true, canShare: true, canTransfer: true, canUpdate: true,
                    canUse: true, canReact: true,
                }},
                // Regular logged-in user with public content
                { state: { isAdmin: false, isDeleted: false, isLoggedIn: true, isPublic: true }, expectedPermissions: {
                    canBookmark: true, canComment: true, canConnect: true, canCopy: true,
                    canDelete: false, canDisconnect: true, canRead: true, canReport: true,
                    canRun: true, canShare: true, canTransfer: false, canUpdate: false,
                    canUse: true, canReact: true,
                }},
                // Anonymous user with public content
                { state: { isAdmin: false, isDeleted: false, isLoggedIn: false, isPublic: true }, expectedPermissions: {
                    canBookmark: false, canComment: false, canConnect: false, canCopy: false,
                    canDelete: false, canDisconnect: false, canRead: true, canReport: false,
                    canRun: true, canShare: true, canTransfer: false, canUpdate: false,
                    canUse: false, canReact: false,
                }},
                // Deleted content
                { state: { isAdmin: true, isDeleted: true, isLoggedIn: true, isPublic: true }, expectedPermissions: {
                    canBookmark: false, canComment: false, canConnect: false, canCopy: false,
                    canDelete: false, canDisconnect: true, canRead: false, canReport: false,
                    canRun: false, canShare: false, canTransfer: false, canUpdate: false,
                    canUse: false, canReact: false,
                }},
                // Private content for non-admin
                { state: { isAdmin: false, isDeleted: false, isLoggedIn: true, isPublic: false }, expectedPermissions: {
                    canBookmark: false, canComment: false, canConnect: false, canCopy: false,
                    canDelete: false, canDisconnect: true, canRead: false, canReport: false,
                    canRun: false, canShare: false, canTransfer: false, canUpdate: false,
                    canUse: false, canReact: false,
                }},
            ];

            for (const testCase of testCases) {
                const perms = defaultPermissions(testCase.state);
                const actualPermissions = {
                    canBookmark: perms.canBookmark(),
                    canComment: perms.canComment(),
                    canConnect: perms.canConnect(),
                    canCopy: perms.canCopy(),
                    canDelete: perms.canDelete(),
                    canDisconnect: perms.canDisconnect(),
                    canRead: perms.canRead(),
                    canReport: perms.canReport(),
                    canRun: perms.canRun(),
                    canShare: perms.canShare(),
                    canTransfer: perms.canTransfer(),
                    canUpdate: perms.canUpdate(),
                    canUse: perms.canUse(),
                    canReact: perms.canReact(),
                };
                expect(actualPermissions).toEqual(testCase.expectedPermissions);
            }
        });
    });

    describe("edge cases", () => {
        it("should handle all falsy values", () => {
            const perms = defaultPermissions({ isAdmin: false, isDeleted: false, isLoggedIn: false, isPublic: false });
            expect(perms.canRead()).toBe(false);
            expect(perms.canDisconnect()).toBe(false);
        });

        it("should handle all truthy values", () => {
            const perms = defaultPermissions({ isAdmin: true, isDeleted: true, isLoggedIn: true, isPublic: true });
            // Even with all true, deleted status should prevent most actions
            expect(perms.canRead()).toBe(false);
            expect(perms.canDisconnect()).toBe(true); // Only needs login
        });
    });
});