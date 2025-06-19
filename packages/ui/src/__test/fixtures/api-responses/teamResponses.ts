import { type Team, type TeamYou, type Role, type Member, type ResourceList } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse } from "./userResponses.js";

/**
 * API response fixtures for teams
 * These represent what components receive from API calls
 */

/**
 * Mock role for team members
 */
const memberRole: Role = {
    __typename: "Role",
    id: "role_member_123456789",
    name: "Member",
    permissions: ["Read"],
    membersCount: 5,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    translations: [],
};

const adminRole: Role = {
    __typename: "Role",
    id: "role_admin_123456789",
    name: "Admin",
    permissions: ["Read", "Write", "Delete", "Invite", "Remove", "Admin"],
    membersCount: 2,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    translations: [],
};

/**
 * Mock member data
 */
const teamMember: Member = {
    __typename: "Member",
    id: "member_123456789012345",
    isAccepted: true,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    user: minimalUserResponse,
    team: null as any, // Avoid circular reference
    roles: [memberRole],
    you: {
        __typename: "MemberYou",
        canDelete: false,
        canUpdate: false,
    },
};

const teamAdmin: Member = {
    __typename: "Member",
    id: "member_admin_123456789",
    isAccepted: true,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    user: completeUserResponse,
    team: null as any, // Avoid circular reference
    roles: [adminRole],
    you: {
        __typename: "MemberYou",
        canDelete: true,
        canUpdate: true,
    },
};

/**
 * Minimal team API response
 */
export const minimalTeamResponse: Team = {
    __typename: "Team",
    id: "team_123456789012345678",
    handle: "test-team",
    name: "Test Team",
    isPrivate: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    bannerImage: null,
    profileImage: null,
    membersCount: 3,
    permissions: JSON.stringify(["Read"]),
    translations: [],
    you: {
        __typename: "TeamYou",
        canDelete: false,
        canRead: true,
        canUpdate: false,
        canBookmark: true,
        canReact: true,
        canReport: true,
        isBookmarked: false,
        reaction: null,
        isReported: false,
    },
};

/**
 * Complete team API response with all fields
 */
export const completeTeamResponse: Team = {
    __typename: "Team",
    id: "team_987654321098765432",
    handle: "awesome-team",
    name: "Awesome Team",
    isPrivate: false,
    createdAt: "2023-06-15T10:30:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    bannerImage: "https://example.com/team-banner.jpg",
    profileImage: "https://example.com/team-logo.png",
    membersCount: 25,
    permissions: JSON.stringify(["Read", "Write", "Delete"]),
    translations: [
        {
            __typename: "TeamTranslation",
            id: "teamtrans_123456789012345",
            language: "en",
            bio: "We build amazing AI-powered tools for developers",
        },
        {
            __typename: "TeamTranslation",
            id: "teamtrans_987654321098765",
            language: "es",
            bio: "Construimos herramientas increíbles impulsadas por IA para desarrolladores",
        },
    ],
    members: [teamAdmin, teamMember],
    roles: [adminRole, memberRole],
    resourceLists: [],
    you: {
        __typename: "TeamYou",
        canDelete: false,
        canRead: true,
        canUpdate: true, // User is admin
        canBookmark: true,
        canReact: true,
        canReport: false, // Can't report own team
        isBookmarked: true,
        reaction: "like",
        isReported: false,
    },
};

/**
 * Private team response
 */
export const privateTeamResponse: Team = {
    __typename: "Team",
    id: "team_private_123456789",
    handle: "private-team",
    name: "Private Team",
    isPrivate: true,
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    bannerImage: null,
    profileImage: null,
    membersCount: 10,
    permissions: JSON.stringify(["Read"]),
    translations: [],
    you: {
        __typename: "TeamYou",
        canDelete: false,
        canRead: true, // User is member
        canUpdate: false,
        canBookmark: false, // Can't bookmark private teams unless member
        canReact: false,
        canReport: false,
        isBookmarked: false,
        reaction: null,
        isReported: false,
    },
};

/**
 * Team with pending membership
 */
export const pendingMembershipTeamResponse: Team = {
    __typename: "Team",
    id: "team_pending_123456789",
    handle: "join-us-team",
    name: "Join Us Team",
    isPrivate: false,
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    bannerImage: null,
    profileImage: "https://example.com/team-avatar.png",
    membersCount: 15,
    permissions: JSON.stringify([]),
    translations: [],
    members: [
        {
            ...teamMember,
            id: "member_pending_123456789",
            isAccepted: false, // Pending invitation
            user: currentUserResponse,
        },
    ],
    you: {
        __typename: "TeamYou",
        canDelete: false,
        canRead: false, // Limited access until accepted
        canUpdate: false,
        canBookmark: true,
        canReact: true,
        canReport: true,
        isBookmarked: false,
        reaction: null,
        isReported: false,
    },
};

// Import current user for membership testing
import { currentUserResponse } from "./userResponses.js";

/**
 * Team variant states for testing
 */
export const teamResponseVariants = {
    minimal: minimalTeamResponse,
    complete: completeTeamResponse,
    private: privateTeamResponse,
    pendingMembership: pendingMembershipTeamResponse,
    ownedByUser: {
        ...completeTeamResponse,
        id: "team_owned_123456789",
        handle: "my-team",
        name: "My Team",
        you: {
            ...completeTeamResponse.you,
            canDelete: true, // Owner can delete
            canUpdate: true,
            canReport: false, // Can't report own team
        },
    },
    largeTeam: {
        ...minimalTeamResponse,
        id: "team_large_123456789",
        handle: "mega-team",
        name: "Mega Team",
        membersCount: 500,
    },
} as const;

/**
 * Team search response
 */
export const teamSearchResponse = {
    __typename: "TeamSearchResult",
    edges: [
        {
            __typename: "TeamEdge",
            cursor: "cursor_1",
            node: teamResponseVariants.complete,
        },
        {
            __typename: "TeamEdge",
            cursor: "cursor_2",
            node: teamResponseVariants.minimal,
        },
        {
            __typename: "TeamEdge",
            cursor: "cursor_3",
            node: teamResponseVariants.private,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: true,
        hasPreviousPage: false,
        startCursor: "cursor_1",
        endCursor: "cursor_3",
    },
};

/**
 * Loading and error states for UI testing
 */
export const teamUIStates = {
    loading: null,
    error: {
        code: "TEAM_NOT_FOUND",
        message: "The requested team could not be found",
    },
    accessDenied: {
        code: "TEAM_ACCESS_DENIED",
        message: "You don't have permission to view this team",
    },
    empty: {
        edges: [],
        pageInfo: {
            __typename: "PageInfo",
            hasNextPage: false,
            hasPreviousPage: false,
            startCursor: null,
            endCursor: null,
        },
    },
};