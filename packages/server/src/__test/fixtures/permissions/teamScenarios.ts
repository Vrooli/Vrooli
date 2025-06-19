import { MemberRole, generatePK } from "@vrooli/shared";
import { standardUser, premiumUser, adminUser } from "./userPersonas.js";
import { type AuthenticatedSessionData } from "../../../types.js";

/**
 * Team permission scenarios for testing team-based access control
 */

export interface TeamScenario {
    team: {
        id: string;
        name: string;
        handle: string;
        isOpenToJoinRequests: boolean;
        isPrivate: boolean;
    };
    members: Array<{
        user: AuthenticatedSessionData;
        role: MemberRole;
        permissions: string[];
    }>;
    description: string;
}

/**
 * Basic team with owner, admin, and member
 */
export const basicTeamScenario: TeamScenario = {
    team: {
        id: "team_10000000000000001",
        name: "Basic Team",
        handle: "basic-team",
        isOpenToJoinRequests: true,
        isPrivate: false,
    },
    members: [
        {
            user: premiumUser,
            role: MemberRole.Owner,
            permissions: ["full_access"],
        },
        {
            user: adminUser,
            role: MemberRole.Admin,
            permissions: ["manage_members", "manage_content"],
        },
        {
            user: standardUser,
            role: MemberRole.Member,
            permissions: ["view_content", "create_content"],
        },
    ],
    description: "Standard team hierarchy for testing role-based permissions",
};

/**
 * Private team with restricted access
 */
export const privateTeamScenario: TeamScenario = {
    team: {
        id: "team_20000000000000002",
        name: "Private Research Team",
        handle: "private-research",
        isOpenToJoinRequests: false,
        isPrivate: true,
    },
    members: [
        {
            user: adminUser,
            role: MemberRole.Owner,
            permissions: ["full_access"],
        },
        {
            user: premiumUser,
            role: MemberRole.Member,
            permissions: ["view_content", "create_content", "view_members"],
        },
    ],
    description: "Private team for testing restricted visibility",
};

/**
 * Large team with multiple admins
 */
export const largeTeamScenario: TeamScenario = {
    team: {
        id: "team_30000000000000003",
        name: "Large Organization",
        handle: "large-org",
        isOpenToJoinRequests: true,
        isPrivate: false,
    },
    members: [
        {
            user: createTeamUser("owner"),
            role: MemberRole.Owner,
            permissions: ["full_access"],
        },
        {
            user: createTeamUser("admin1"),
            role: MemberRole.Admin,
            permissions: ["manage_members", "manage_content", "manage_settings"],
        },
        {
            user: createTeamUser("admin2"),
            role: MemberRole.Admin,
            permissions: ["manage_members", "manage_content"],
        },
        {
            user: createTeamUser("member1"),
            role: MemberRole.Member,
            permissions: ["view_content", "create_content"],
        },
        {
            user: createTeamUser("member2"),
            role: MemberRole.Member,
            permissions: ["view_content"],
        },
    ],
    description: "Large team for testing scalability and multiple admin scenarios",
};

/**
 * Team with pending invitations
 */
export const invitationTeamScenario: TeamScenario = {
    team: {
        id: "team_40000000000000004",
        name: "Invitation Test Team",
        handle: "invite-team",
        isOpenToJoinRequests: false,
        isPrivate: false,
    },
    members: [
        {
            user: standardUser,
            role: MemberRole.Owner,
            permissions: ["full_access"],
        },
    ],
    description: "Team for testing invitation flows",
};

/**
 * Helper to create a team user
 */
function createTeamUser(suffix: string): AuthenticatedSessionData {
    return {
        ...standardUser,
        id: generatePK().toString(),
        handle: `team_user_${suffix}`,
        name: `Team User ${suffix}`,
        email: `team_${suffix}@example.com`,
    };
}

/**
 * Create a custom team scenario
 */
export function createTeamScenario(
    teamOverrides: Partial<TeamScenario["team"]>,
    members: Array<{ userId: string; role: MemberRole }>,
): TeamScenario {
    const teamId = teamOverrides.id || `team_${generatePK()}`;
    
    return {
        team: {
            id: teamId,
            name: "Custom Team",
            handle: "custom-team",
            isOpenToJoinRequests: true,
            isPrivate: false,
            ...teamOverrides,
        },
        members: members.map(({ userId, role }) => ({
            user: createTeamUser(userId),
            role,
            permissions: getDefaultPermissionsForRole(role),
        })),
        description: "Custom team scenario",
    };
}

/**
 * Get default permissions for a role
 */
function getDefaultPermissionsForRole(role: MemberRole): string[] {
    switch (role) {
        case MemberRole.Owner:
            return ["full_access"];
        case MemberRole.Admin:
            return ["manage_members", "manage_content"];
        case MemberRole.Member:
            return ["view_content", "create_content"];
        default:
            return ["view_content"];
    }
}

/**
 * Nested team hierarchy scenario - for organizations with sub-teams
 */
export const nestedTeamHierarchyScenario = {
    parentTeam: {
        id: "team_80000000000000008",
        name: "Parent Organization",
        handle: "parent-org",
        isOpenToJoinRequests: false,
        isPrivate: false,
    },
    subTeams: [
        {
            team: {
                id: "team_81000000000000001",
                name: "Engineering Sub-Team",
                handle: "eng-team",
                parentId: "team_80000000000000008",
                isPrivate: false,
            },
            members: [
                {
                    user: createTeamUser("eng-lead"),
                    role: MemberRole.Admin,
                    permissions: ["manage_content", "view_members"],
                },
            ],
        },
        {
            team: {
                id: "team_82000000000000002",
                name: "Design Sub-Team",
                handle: "design-team",
                parentId: "team_80000000000000008",
                isPrivate: false,
            },
            members: [
                {
                    user: createTeamUser("design-lead"),
                    role: MemberRole.Admin,
                    permissions: ["manage_content", "view_members"],
                },
            ],
        },
    ],
    parentMembers: [
        {
            user: adminUser,
            role: MemberRole.Owner,
            permissions: ["full_access", "manage_subteams"],
        },
        {
            user: premiumUser,
            role: MemberRole.Admin,
            permissions: ["manage_members", "view_subteams"],
        },
    ],
    description: "Hierarchical team structure for testing inherited permissions",
};

/**
 * Cross-team permission scenarios
 */
export const crossTeamScenarios = {
    /**
     * User is owner of one team, member of another
     */
    multipleRoles: {
        user: standardUser,
        teams: [
            { teamId: "team_50000000000000005", role: MemberRole.Owner },
            { teamId: "team_60000000000000006", role: MemberRole.Member },
        ],
        description: "User with different roles across teams",
    },
    
    /**
     * User trying to access team they're not part of
     */
    noAccess: {
        user: standardUser,
        targetTeamId: "team_70000000000000007",
        description: "User with no access to target team",
    },
};