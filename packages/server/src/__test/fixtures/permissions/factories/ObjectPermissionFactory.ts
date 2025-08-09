/**
 * Object Permission Factory
 * 
 * Generic factory for creating permission fixtures for any Vrooli object type.
 * This factory generates comprehensive permission scenarios for testing
 * access control across all 47+ object types.
 */

import { generatePK, type VisibilityType } from "@vrooli/shared";
import { type SessionData } from "../../../../types.js";
import { type ApiKeyAuthData, type PermissionScenario } from "../types.js";
import { ApiKeyFactory } from "./ApiKeyFactory.js";
import { UserSessionFactory } from "./UserSessionFactory.js";

/**
 * Configuration for generating object permission fixtures
 */
export interface ObjectPermissionConfig<TObject> {
    /** The object type name (e.g., "Bookmark", "Project") */
    objectType: string;

    /** Function to create a minimal object */
    createMinimal: (overrides?: Partial<TObject>) => TObject;

    /** Function to create a complete object */
    createComplete: (overrides?: Partial<TObject>) => TObject;

    /** Supported actions for this object type */
    supportedActions: string[];

    /** Whether this object can be team-owned */
    canBeTeamOwned?: boolean;

    /** Whether this object has visibility settings */
    hasVisibility?: boolean;

    /** Custom permission rules */
    customRules?: {
        [action: string]: (session: SessionData | ApiKeyAuthData, object: TObject) => boolean;
    };
}

/**
 * Generic factory for creating object-specific permission fixtures
 */
export class ObjectPermissionFactory<TObject extends { id?: string; __typename?: string }> {
    private readonly userFactory = new UserSessionFactory();
    private readonly apiKeyFactory = new ApiKeyFactory();

    constructor(private readonly config: ObjectPermissionConfig<TObject>) { }

    /**
     * Create a public object owned by a user
     */
    createPublicUserOwned(userId = "222222222222222222"): PermissionScenario<TObject> {
        const object = this.config.createComplete({
            id: generatePK().toString(),
            __typename: this.config.objectType,
            owner: { id: userId },
            isPublic: true,
        } as unknown as Partial<TObject>);

        return {
            id: `${this.config.objectType.toLowerCase()}_public_user_owned`,
            description: `Public ${this.config.objectType} owned by user`,
            resource: object,
            actors: this.generateStandardActors(userId),
            actions: this.config.supportedActions,
        };
    }

    /**
     * Create a private object owned by a user
     */
    createPrivateUserOwned(userId = "222222222222222222"): PermissionScenario<TObject> {
        const object = this.config.createComplete({
            id: generatePK().toString(),
            __typename: this.config.objectType,
            owner: { id: userId },
            isPublic: false,
        } as unknown as Partial<TObject>);

        return {
            id: `${this.config.objectType.toLowerCase()}_private_user_owned`,
            description: `Private ${this.config.objectType} owned by user`,
            resource: object,
            actors: this.generateStandardActors(userId),
            actions: this.config.supportedActions,
        };
    }

    /**
     * Create a team-owned object
     */
    createTeamOwned(teamId?: string): PermissionScenario<TObject> | null {
        if (!this.config.canBeTeamOwned) {
            return null;
        }

        const teamIdString = teamId || generatePK().toString();
        const object = this.config.createComplete({
            id: generatePK().toString(),
            __typename: this.config.objectType,
            team: { id: teamIdString },
            isPublic: false,
        } as unknown as Partial<TObject>);
        const owner = this.userFactory.withTeam(
            this.userFactory.createSession({ id: "111111111111111111" }),
            teamIdString,
            "Owner",
        );

        const admin = this.userFactory.withTeam(
            this.userFactory.createSession({ id: "222222222222222222" }),
            teamIdString,
            "Admin",
        );

        const member = this.userFactory.withTeam(
            this.userFactory.createSession({ id: "333333333333333333" }),
            teamIdString,
            "Member",
        );

        const nonMember = this.userFactory.createSession({ id: "444444444444444444" });

        return {
            id: `${this.config.objectType.toLowerCase()}_team_owned`,
            description: `Team-owned ${this.config.objectType}`,
            resource: object,
            actors: [
                {
                    id: "team_owner",
                    session: owner,
                    permissions: this.generateTeamOwnerPermissions(),
                },
                {
                    id: "team_admin",
                    session: admin,
                    permissions: this.generateTeamAdminPermissions(),
                },
                {
                    id: "team_member",
                    session: member,
                    permissions: this.generateTeamMemberPermissions(),
                },
                {
                    id: "non_member",
                    session: nonMember,
                    permissions: this.generateNoAccessPermissions(),
                },
            ],
            actions: this.config.supportedActions,
        };
    }

    /**
     * Create an unlisted object (visible via direct link)
     */
    createUnlisted(userId = "222222222222222222"): PermissionScenario<TObject> | null {
        if (!this.config.hasVisibility) {
            return null;
        }

        const object = this.config.createComplete({
            id: generatePK().toString(),
            __typename: this.config.objectType,
            owner: { id: userId },
            visibility: "Unlisted" as VisibilityType,
        } as unknown as Partial<TObject>);

        return {
            id: `${this.config.objectType.toLowerCase()}_unlisted`,
            description: `Unlisted ${this.config.objectType} (visible via direct link)`,
            resource: object,
            actors: this.generateStandardActors(userId),
            actions: this.config.supportedActions,
        };
    }

    /**
     * Create comprehensive test suite for the object type
     */
    createTestSuite(): PermissionScenario<TObject>[] {
        const scenarios: PermissionScenario<TObject>[] = [];

        // User-owned scenarios
        scenarios.push(this.createPublicUserOwned());
        scenarios.push(this.createPrivateUserOwned());

        // Team-owned scenario
        const teamScenario = this.createTeamOwned();
        if (teamScenario) {
            scenarios.push(teamScenario);
        }

        // Visibility scenario
        const unlistedScenario = this.createUnlisted();
        if (unlistedScenario) {
            scenarios.push(unlistedScenario);
        }

        // API key scenarios
        scenarios.push(this.createApiKeyScenario());

        // Edge cases
        scenarios.push(this.createDeletedOwnerScenario());
        scenarios.push(this.createSuspendedOwnerScenario());

        return scenarios;
    }

    /**
     * Generate standard actors for testing
     */
    private generateStandardActors(ownerId: string) {
        const owner = this.userFactory.createSession({ id: ownerId });
        const otherUser = this.userFactory.createStandard();
        const admin = this.userFactory.createAdmin();
        const guest = this.userFactory.createGuest();
        const banned = this.userFactory.createBanned();

        return [
            {
                id: "owner",
                session: owner,
                permissions: this.generateOwnerPermissions(),
            },
            {
                id: "other_user",
                session: otherUser,
                permissions: this.generateOtherUserPermissions(),
            },
            {
                id: "admin",
                session: admin,
                permissions: this.generateAdminPermissions(),
            },
            {
                id: "guest",
                session: guest,
                permissions: this.generateGuestPermissions(),
            },
            {
                id: "banned",
                session: banned,
                permissions: this.generateNoAccessPermissions(),
            },
        ];
    }

    /**
     * Create API key access scenario
     */
    private createApiKeyScenario(): PermissionScenario<TObject> {
        const userId = "222222222222222222";
        const object = this.config.createComplete({
            id: generatePK().toString(),
            __typename: this.config.objectType,
            owner: { id: userId },
            isPublic: true,
        } as unknown as Partial<TObject>);

        const readOnlyKey = this.apiKeyFactory.createReadOnlyPublic({ userId });
        const writeKey = this.apiKeyFactory.createWrite({ userId });
        const otherUserKey = this.apiKeyFactory.createWrite({ userId: "333333333333333333" });

        return {
            id: `${this.config.objectType.toLowerCase()}_api_key_access`,
            description: `API key access to ${this.config.objectType}`,
            resource: object,
            actors: [
                {
                    id: "read_only_key",
                    session: readOnlyKey,
                    permissions: {
                        read: true,
                        create: false,
                        update: false,
                        delete: false,
                    },
                },
                {
                    id: "write_key",
                    session: writeKey,
                    permissions: {
                        read: true,
                        create: true,
                        update: true,
                        delete: true,
                    },
                },
                {
                    id: "other_user_key",
                    session: otherUserKey,
                    permissions: {
                        read: true, // Public object
                        create: false,
                        update: false,
                        delete: false,
                    },
                },
            ],
            actions: this.config.supportedActions,
        };
    }

    /**
     * Create deleted owner scenario
     */
    private createDeletedOwnerScenario(): PermissionScenario<TObject> {
        const object = this.config.createComplete({
            id: generatePK().toString(),
            __typename: this.config.objectType,
            owner: { id: "999999999999999999" }, // Use a valid string ID for deleted user
            isPublic: true,
        } as unknown as Partial<TObject>);

        const admin = this.userFactory.createAdmin();
        const user = this.userFactory.createStandard();

        return {
            id: `${this.config.objectType.toLowerCase()}_deleted_owner`,
            description: `${this.config.objectType} with deleted owner`,
            resource: object,
            actors: [
                {
                    id: "admin",
                    session: admin,
                    permissions: this.generateAdminPermissions(),
                },
                {
                    id: "regular_user",
                    session: user,
                    permissions: {
                        read: true, // Public
                        create: false,
                        update: false,
                        delete: false,
                    },
                },
            ],
            actions: this.config.supportedActions,
        };
    }

    /**
     * Create suspended owner scenario
     */
    private createSuspendedOwnerScenario(): PermissionScenario<TObject> {
        const suspendedUser = this.userFactory.createSuspended();
        const object = this.config.createComplete({
            id: generatePK().toString(),
            __typename: this.config.objectType,
            owner: { id: suspendedUser.id },
            isPublic: false,
        } as unknown as Partial<TObject>);

        const admin = this.userFactory.createAdmin();

        return {
            id: `${this.config.objectType.toLowerCase()}_suspended_owner`,
            description: `${this.config.objectType} owned by suspended user`,
            resource: object,
            actors: [
                {
                    id: "suspended_owner",
                    session: suspendedUser,
                    permissions: this.generateNoAccessPermissions(), // Suspended users can't access
                },
                {
                    id: "admin",
                    session: admin,
                    permissions: this.generateAdminPermissions(),
                },
            ],
            actions: this.config.supportedActions,
        };
    }

    // Permission generators for different roles
    private generateOwnerPermissions(): Record<string, boolean> {
        const perms: Record<string, boolean> = {};
        for (const action of this.config.supportedActions) {
            perms[action] = true;
        }
        return perms;
    }

    private generateAdminPermissions(): Record<string, boolean> {
        return this.generateOwnerPermissions(); // Admins can do everything
    }

    private generateOtherUserPermissions(): Record<string, boolean> {
        const perms: Record<string, boolean> = {};
        for (const action of this.config.supportedActions) {
            perms[action] = action === "read"; // Only read public objects
        }
        return perms;
    }

    private generateGuestPermissions(): Record<string, boolean> {
        const perms: Record<string, boolean> = {};
        for (const action of this.config.supportedActions) {
            perms[action] = action === "read"; // Only read public objects
        }
        return perms;
    }

    private generateNoAccessPermissions(): Record<string, boolean> {
        const perms: Record<string, boolean> = {};
        for (const action of this.config.supportedActions) {
            perms[action] = false;
        }
        return perms;
    }

    private generateTeamOwnerPermissions(): Record<string, boolean> {
        return this.generateOwnerPermissions();
    }

    private generateTeamAdminPermissions(): Record<string, boolean> {
        return this.generateOwnerPermissions();
    }

    private generateTeamMemberPermissions(): Record<string, boolean> {
        const perms: Record<string, boolean> = {};
        for (const action of this.config.supportedActions) {
            perms[action] = ["read", "create"].includes(action);
        }
        return perms;
    }
}
