/**
 * User Session Factory
 * 
 * Factory for creating user session fixtures with various permission levels
 * and authentication states.
 */

import { AccountStatus, generatePK } from "@vrooli/shared";
import { type TestSessionData } from "../types.js";
import { BasePermissionFactory } from "./BasePermissionFactory.js";

/**
 * Factory for creating user session fixtures
 */
export class UserSessionFactory extends BasePermissionFactory<TestSessionData> {

    /**
     * ID counter for generating consistent test IDs
     */
    private idCounter = 0;

    /**
     * Create a session with specific overrides
     */
    createSession(overrides?: Partial<TestSessionData>): TestSessionData {
        this.idCounter++;

        const baseSession: TestSessionData = {
            ...this.baseUserData,
            id: this.generateTestId("", this.idCounter),
            handle: `user${this.idCounter}`,
            name: `Test User ${this.idCounter}`,
            email: `user${this.idCounter}@test.com`,
            emailVerified: true,
            accountStatus: AccountStatus.Unlocked,
            isLoggedIn: true,
            timeZone: this.defaultOptions.timeZone ?? "UTC",
            hasPremium: false,
            roles: [],
        };

        return this.mergeWithDefaults(baseSession, overrides);
    }

    /**
     * Create an admin user session
     */
    createAdmin(overrides?: Partial<TestSessionData>): TestSessionData {
        return this.createSession({
            id: generatePK().toString(),
            handle: "admin",
            name: "System Admin",
            email: "admin@vrooli.com",
            hasPremium: true,
            roles: [{
                role: {
                    name: "Admin",
                    permissions: JSON.stringify(["*"]),
                },
            }],
            ...overrides,
        });
    }

    /**
     * Create a standard user session
     */
    createStandard(overrides?: Partial<TestSessionData>): TestSessionData {
        return this.createSession({
            id: generatePK().toString(),
            handle: "johndoe",
            name: "John Doe",
            email: "john@example.com",
            ...overrides,
        });
    }

    /**
     * Create a premium user session
     */
    createPremium(overrides?: Partial<TestSessionData>): TestSessionData {
        return this.createSession({
            id: generatePK().toString(),
            handle: "premiumuser",
            name: "Premium User",
            email: "premium@example.com",
            hasPremium: true,
            ...overrides,
        });
    }

    /**
     * Create an unverified user session
     */
    createUnverified(overrides?: Partial<TestSessionData>): TestSessionData {
        return this.createSession({
            id: generatePK().toString(),
            handle: "unverified",
            name: "Unverified User",
            email: "unverified@example.com",
            emailVerified: false,
            ...overrides,
        });
    }

    /**
     * Create a banned user session
     */
    createBanned(overrides?: Partial<TestSessionData>): TestSessionData {
        return this.createSession({
            id: generatePK().toString(),
            handle: "banned",
            name: "Banned User",
            email: "banned@example.com",
            accountStatus: AccountStatus.SoftLocked,
            ...overrides,
        });
    }

    /**
     * Create a suspended user session
     */
    createSuspended(overrides?: Partial<TestSessionData>): TestSessionData {
        return this.createSession({
            id: generatePK().toString(),
            handle: "suspended",
            name: "Suspended User",
            email: "suspended@example.com",
            accountStatus: AccountStatus.HardLocked,
            ...overrides,
        });
    }

    /**
     * Create a bot user session
     */
    createBot(overrides?: Partial<TestSessionData>): TestSessionData {
        return this.createSession({
            id: generatePK().toString(),
            handle: "testbot",
            name: "Test Bot",
            email: "bot@vrooli.com",
            isBot: true,
            ...overrides,
        });
    }

    /**
     * Create a guest user session (not logged in)
     */
    createGuest(overrides?: Partial<TestSessionData>): TestSessionData {
        return {
            ...this.baseUserData,
            id: "",
            accountStatus: AccountStatus.Unlocked,
            isLoggedIn: false,
            timeZone: this.defaultOptions.timeZone ?? "UTC",
            ...overrides,
        } as TestSessionData;
    }

    /**
     * Create a user with custom roles
     */
    createWithCustomRole(
        roleName: string,
        permissions: string[],
        overrides?: Partial<TestSessionData>,
    ): TestSessionData {
        return this.createSession({
            id: generatePK().toString(),
            handle: "customrole",
            name: "Custom Role User",
            email: "custom@example.com",
            roles: [{
                role: {
                    name: roleName,
                    permissions: JSON.stringify(permissions),
                },
            }],
            ...overrides,
        });
    }

    /**
     * Create a user with expired premium
     */
    createExpiredPremium(overrides?: Partial<TestSessionData>): TestSessionData {
        return this.createSession({
            id: generatePK().toString(),
            handle: "expiredpremium",
            name: "Expired Premium User",
            email: "expired@example.com",
            hasPremium: false,
            _testPreviouslyPremium: true,
            ...overrides,
        });
    }

    /**
     * Create a user with partial data (edge case)
     */
    createPartial(overrides?: Partial<TestSessionData>): TestSessionData {
        return {
            ...this.baseUserData,
            id: generatePK().toString(),
            isLoggedIn: true,
            timeZone: this.defaultOptions.timeZone ?? "UTC",
            ...overrides,
        } as TestSessionData;
    }

    /**
     * Create a user with conflicting permissions
     */
    createConflictingPermissions(overrides?: Partial<TestSessionData>): TestSessionData {
        return this.createSession({
            id: generatePK().toString(),
            handle: "conflicted",
            name: "Conflicted User",
            email: "conflict@example.com",
            roles: [
                {
                    role: {
                        name: "Viewer",
                        permissions: JSON.stringify(["content.read"]),
                    },
                },
                {
                    role: {
                        name: "Editor",
                        permissions: JSON.stringify(["content.write", "!content.read"]),
                    },
                },
            ],
            ...overrides,
        });
    }

    /**
     * Create a user with all possible permissions
     */
    createMaxPermissions(overrides?: Partial<TestSessionData>): TestSessionData {
        return this.createSession({
            id: generatePK().toString(),
            handle: "maxperms",
            name: "Max Permissions User",
            email: "max@example.com",
            hasPremium: true,
            roles: [
                {
                    role: {
                        name: "SuperAdmin",
                        permissions: JSON.stringify(["*"]),
                    },
                },
            ],
            ...overrides,
        });
    }

    /**
     * Create a user in the process of deletion
     */
    createDeleting(overrides?: Partial<TestSessionData>): TestSessionData {
        return this.createSession({
            id: generatePK().toString(),
            handle: "deleting",
            name: "Deleting User",
            email: "deleting@example.com",
            accountStatus: AccountStatus.Deleted,
            ...overrides,
        });
    }

    /**
     * Create multiple users with relationships
     */
    createTeam(teamSize = 3): TestSessionData[] {
        const team: TestSessionData[] = [];
        const teamId = generatePK().toString();

        // Owner
        team.push(this.withTeam(
            this.createSession({
                handle: "owner",
                name: "Team Owner",
            }),
            teamId,
            "Owner",
        ));

        // Admin
        if (teamSize > 1) {
            team.push(this.withTeam(
                this.createSession({
                    handle: "admin",
                    name: "Team Admin",
                }),
                teamId,
                "Admin",
            ));
        }

        // Members
        for (let i = 2; i < teamSize; i++) {
            team.push(this.withTeam(
                this.createSession({
                    handle: `member${i - 1}`,
                    name: `Team Member ${i - 1}`,
                }),
                teamId,
                "Member",
            ));
        }

        return team;
    }
}
