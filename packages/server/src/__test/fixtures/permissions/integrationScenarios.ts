import { MemberRole } from "@vrooli/shared";
import { adminUser, standardUser, premiumUser, bannedUser, guestUser } from "./userPersonas.js";
import { readOnlyPublicApiKey, writeApiKey, botApiKey, type ApiKeyFullAuthData } from "./apiKeyPermissions.js";
import { basicTeamScenario, privateTeamScenario } from "./teamScenarios.js";
import { type AuthenticatedSessionData } from "../../../types.js";

/**
 * Complex integration scenarios that combine multiple permission concepts
 */

export interface IntegrationScenario {
    name: string;
    description: string;
    actors: Array<{
        type: "user" | "apiKey";
        data: AuthenticatedSessionData | ApiKeyFullAuthData;
        role?: string;
    }>;
    resource: {
        type: string;
        id: string;
        ownerId?: string;
        teamId?: string;
        isPrivate?: boolean;
    };
    expectedOutcomes: Record<string, {
        canRead: boolean;
        canWrite: boolean;
        canDelete: boolean;
        reason: string;
    }>;
}

/**
 * Scenario 1: Public resource access hierarchy
 */
export const publicResourceScenario: IntegrationScenario = {
    name: "Public Resource Access",
    description: "Testing access to a public resource by different actors",
    actors: [
        { type: "user", data: adminUser, role: "Admin" },
        { type: "user", data: standardUser, role: "Owner" },
        { type: "user", data: premiumUser, role: "Viewer" },
        { type: "user", data: guestUser, role: "Guest" },
        { type: "apiKey", data: readOnlyPublicApiKey, role: "API_ReadOnly" },
    ],
    resource: {
        type: "Project",
        id: "project_10000000000000001",
        ownerId: standardUser.id,
        isPrivate: false,
    },
    expectedOutcomes: {
        Admin: {
            canRead: true,
            canWrite: true,
            canDelete: true,
            reason: "Admins have full access to all resources",
        },
        Owner: {
            canRead: true,
            canWrite: true,
            canDelete: true,
            reason: "Owners have full control of their resources",
        },
        Viewer: {
            canRead: true,
            canWrite: false,
            canDelete: false,
            reason: "Public resources are readable by all authenticated users",
        },
        Guest: {
            canRead: true,
            canWrite: false,
            canDelete: false,
            reason: "Public resources are readable even by guests",
        },
        API_ReadOnly: {
            canRead: true,
            canWrite: false,
            canDelete: false,
            reason: "Read-only API keys can access public resources",
        },
    },
};

/**
 * Scenario 2: Private team resource with role hierarchy
 */
export const teamResourceScenario: IntegrationScenario = {
    name: "Team Resource Permissions",
    description: "Testing team member access with different roles",
    actors: [
        { type: "user", data: premiumUser, role: "TeamOwner" },
        { type: "user", data: adminUser, role: "TeamAdmin" },
        { type: "user", data: standardUser, role: "TeamMember" },
        { type: "user", data: bannedUser, role: "NonMember" },
        { type: "apiKey", data: writeApiKey, role: "MemberAPI" },
    ],
    resource: {
        type: "Routine",
        id: "routine_20000000000000002",
        teamId: basicTeamScenario.team.id,
        isPrivate: true,
    },
    expectedOutcomes: {
        TeamOwner: {
            canRead: true,
            canWrite: true,
            canDelete: true,
            reason: "Team owners have full access to team resources",
        },
        TeamAdmin: {
            canRead: true,
            canWrite: true,
            canDelete: false,
            reason: "Team admins can modify but not delete resources",
        },
        TeamMember: {
            canRead: true,
            canWrite: true,
            canDelete: false,
            reason: "Team members can view and contribute to resources",
        },
        NonMember: {
            canRead: false,
            canWrite: false,
            canDelete: false,
            reason: "Non-members cannot access private team resources",
        },
        MemberAPI: {
            canRead: true,
            canWrite: true,
            canDelete: false,
            reason: "API key inherits user's team permissions",
        },
    },
};

/**
 * Scenario 3: Cross-team collaboration
 */
export const crossTeamCollaborationScenario: IntegrationScenario = {
    name: "Cross-Team Collaboration",
    description: "Testing access when resource is shared between teams",
    actors: [
        { type: "user", data: createCollabUser("team1_owner"), role: "Team1Owner" },
        { type: "user", data: createCollabUser("team2_member"), role: "Team2Member" },
        { type: "user", data: createCollabUser("both_teams"), role: "BothTeamsMember" },
        { type: "user", data: createCollabUser("neither_team"), role: "Outsider" },
    ],
    resource: {
        type: "Project",
        id: "project_30000000000000003",
        // Shared between two teams
        teamId: "shared_between_teams",
        isPrivate: true,
    },
    expectedOutcomes: {
        Team1Owner: {
            canRead: true,
            canWrite: true,
            canDelete: true,
            reason: "Owner of collaborating team has full access",
        },
        Team2Member: {
            canRead: true,
            canWrite: true,
            canDelete: false,
            reason: "Members of collaborating teams can contribute",
        },
        BothTeamsMember: {
            canRead: true,
            canWrite: true,
            canDelete: false,
            reason: "Member of multiple collaborating teams",
        },
        Outsider: {
            canRead: false,
            canWrite: false,
            canDelete: false,
            reason: "Not part of any collaborating team",
        },
    },
};

/**
 * Scenario 4: Bot automation permissions
 */
export const botAutomationScenario: IntegrationScenario = {
    name: "Bot Automation Access",
    description: "Testing bot API keys with various permission levels",
    actors: [
        { type: "apiKey", data: botApiKey, role: "FullBot" },
        { type: "apiKey", data: createBotApiKey("read_only"), role: "ReadBot" },
        { type: "apiKey", data: createBotApiKey("team_bot"), role: "TeamBot" },
        { type: "user", data: standardUser, role: "HumanUser" },
    ],
    resource: {
        type: "Automation",
        id: "automation_40000000000000004",
        ownerId: "bot_owner_id",
        isPrivate: false,
    },
    expectedOutcomes: {
        FullBot: {
            canRead: true,
            canWrite: true,
            canDelete: true,
            reason: "Bot with full permissions can manage automations",
        },
        ReadBot: {
            canRead: true,
            canWrite: false,
            canDelete: false,
            reason: "Read-only bot can monitor but not modify",
        },
        TeamBot: {
            canRead: true,
            canWrite: true,
            canDelete: false,
            reason: "Team bot can execute within team scope",
        },
        HumanUser: {
            canRead: true,
            canWrite: false,
            canDelete: false,
            reason: "Human users can view public automations",
        },
    },
};

/**
 * Scenario 5: Permission escalation attempt
 */
export const escalationAttemptScenario: IntegrationScenario = {
    name: "Permission Escalation Prevention",
    description: "Testing that users cannot escalate their own permissions",
    actors: [
        { type: "user", data: standardUser, role: "RegularUser" },
        { type: "user", data: adminUser, role: "Admin" },
        { type: "apiKey", data: writeApiKey, role: "UserAPI" },
    ],
    resource: {
        type: "Role",
        id: "role_50000000000000005",
        ownerId: "system",
    },
    expectedOutcomes: {
        RegularUser: {
            canRead: false,
            canWrite: false,
            canDelete: false,
            reason: "Regular users cannot modify roles",
        },
        Admin: {
            canRead: true,
            canWrite: true,
            canDelete: true,
            reason: "Only admins can manage roles",
        },
        UserAPI: {
            canRead: false,
            canWrite: false,
            canDelete: false,
            reason: "API keys cannot modify permission structures",
        },
    },
};

/**
 * Helper functions
 */
function createCollabUser(suffix: string): AuthenticatedSessionData {
    return {
        ...standardUser,
        id: `collab_${suffix}_${Date.now()}`,
        handle: `collab_${suffix}`,
        name: `Collab User ${suffix}`,
        email: `collab_${suffix}@example.com`,
    };
}

function createBotApiKey(type: string): ApiKeyFullAuthData {
    return {
        ...botApiKey,
        id: `bot_api_${type}_${Date.now()}`,
        permissions: {
            ...botApiKey.permissions,
            ...(type === "read_only" ? { write: "None" } : {}),
        },
    };
}

/**
 * Generate test cases from scenarios
 */
export function generatePermissionTests(scenario: IntegrationScenario) {
    const tests: Array<{
        actor: string;
        action: "read" | "write" | "delete";
        expected: boolean;
        reason: string;
    }> = [];

    for (const actor of scenario.actors) {
        const outcome = scenario.expectedOutcomes[actor.role!];
        if (!outcome) continue;

        tests.push(
            { actor: actor.role!, action: "read", expected: outcome.canRead, reason: outcome.reason },
            { actor: actor.role!, action: "write", expected: outcome.canWrite, reason: outcome.reason },
            { actor: actor.role!, action: "delete", expected: outcome.canDelete, reason: outcome.reason },
        );
    }

    return tests;
}