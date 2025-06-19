/**
 * API response fixtures for member-related endpoints
 * These represent data as returned from the server API
 */

/**
 * Basic member response (minimal data)
 */
export const minimalMemberResponse = {
    __typename: "Member",
    id: "member_123456789012345678",
    isAdmin: false,
    permissions: ["Read"],
    user: {
        __typename: "User",
        id: "user_123456789012345678",
        handle: "testuser",
        name: "Test User",
        profileImage: null,
        isBot: false,
        updatedAt: "2024-01-15T10:00:00.000Z",
    },
    team: {
        __typename: "Team",
        id: "team_123456789012345678",
        handle: "testteam",
        name: "Test Team",
    },
    createdAt: "2024-01-15T10:00:00.000Z",
    updatedAt: "2024-01-15T10:00:00.000Z",
};

/**
 * Complete member response (all optional fields)
 */
export const completeMemberResponse = {
    __typename: "Member",
    id: "member_987654321098765432",
    isAdmin: true,
    permissions: ["Create", "Read", "Update", "Delete", "UseApi", "Manage"],
    user: {
        __typename: "User",
        id: "user_987654321098765432",
        handle: "adminuser",
        name: "Admin User",
        profileImage: "https://example.com/profile.jpg",
        isBot: false,
        updatedAt: "2024-01-15T11:00:00.000Z",
    },
    team: {
        __typename: "Team",
        id: "team_987654321098765432",
        handle: "adminteam",
        name: "Admin Team",
        profileImage: "https://example.com/team.jpg",
    },
    createdAt: "2024-01-15T09:00:00.000Z",
    updatedAt: "2024-01-15T11:00:00.000Z",
};

/**
 * Member with various permission levels
 */
export const developerMemberResponse = {
    __typename: "Member",
    id: "member_456789123456789012",
    isAdmin: false,
    permissions: ["Create", "Read", "Update", "UseApi"],
    user: {
        __typename: "User",
        id: "user_456789123456789012",
        handle: "developer",
        name: "Jane Developer",
        profileImage: null,
        isBot: false,
        updatedAt: "2024-01-15T12:00:00.000Z",
    },
    team: {
        __typename: "Team",
        id: "team_456789123456789012",
        handle: "devteam",
        name: "Development Team",
    },
    createdAt: "2024-01-15T08:00:00.000Z",
    updatedAt: "2024-01-15T12:00:00.000Z",
};

export const moderatorMemberResponse = {
    __typename: "Member",
    id: "member_789012345678901234",
    isAdmin: false,
    permissions: ["Read", "Update", "Delete"],
    user: {
        __typename: "User",
        id: "user_789012345678901234",
        handle: "moderator",
        name: "Bob Moderator",
        profileImage: null,
        isBot: false,
        updatedAt: "2024-01-15T13:00:00.000Z",
    },
    team: {
        __typename: "Team",
        id: "team_789012345678901234",
        handle: "modteam",
        name: "Moderation Team",
    },
    createdAt: "2024-01-15T07:00:00.000Z",
    updatedAt: "2024-01-15T13:00:00.000Z",
};

/**
 * Member update responses
 */
export const memberUpdateResponse = {
    __typename: "Member",
    id: "member_123456789012345678",
    isAdmin: true, // Changed from false
    permissions: ["Create", "Read", "Update", "Delete", "UseApi", "Manage"], // Expanded permissions
    user: {
        __typename: "User",
        id: "user_123456789012345678",
        handle: "testuser",
        name: "Test User",
        profileImage: null,
        isBot: false,
        updatedAt: "2024-01-15T10:00:00.000Z",
    },
    team: {
        __typename: "Team",
        id: "team_123456789012345678",
        handle: "testteam",
        name: "Test Team",
    },
    createdAt: "2024-01-15T10:00:00.000Z",
    updatedAt: "2024-01-15T14:00:00.000Z", // Updated timestamp
};

/**
 * List responses for multiple members
 */
export const membersListResponse = {
    __typename: "MemberSearchResult",
    edges: [
        {
            cursor: "member_123456789012345678",
            node: minimalMemberResponse,
        },
        {
            cursor: "member_987654321098765432",
            node: completeMemberResponse,
        },
        {
            cursor: "member_456789123456789012",
            node: developerMemberResponse,
        },
        {
            cursor: "member_789012345678901234",
            node: moderatorMemberResponse,
        },
    ],
    pageInfo: {
        hasNextPage: false,
        hasPreviousPage: false,
        startCursor: "member_123456789012345678",
        endCursor: "member_789012345678901234",
    },
    total: 4,
};

/**
 * Error response examples
 */
export const memberNotFoundResponse = {
    errors: [
        {
            message: "Member not found",
            extensions: {
                code: "NOT_FOUND",
                field: "id",
            },
        },
    ],
};

export const memberPermissionDeniedResponse = {
    errors: [
        {
            message: "Permission denied: Cannot modify member permissions",
            extensions: {
                code: "FORBIDDEN",
                requiredPermissions: ["Manage"],
            },
        },
    ],
};

export const memberInvalidPermissionsResponse = {
    errors: [
        {
            message: "Invalid permissions specified",
            extensions: {
                code: "INVALID_INPUT",
                field: "permissions",
                invalidValues: ["InvalidPermission"],
            },
        },
    ],
};

/**
 * Member role examples for common use cases
 */
export const memberRoleExamples = {
    owner: {
        ...completeMemberResponse,
        id: "member_owner_123456789012",
        isAdmin: true,
        permissions: ["Create", "Read", "Update", "Delete", "UseApi", "Manage"],
        user: {
            ...completeMemberResponse.user,
            id: "user_owner_123456789012",
            handle: "teamowner",
            name: "Team Owner",
        },
    },
    
    readOnly: {
        ...minimalMemberResponse,
        id: "member_readonly_123456789012",
        isAdmin: false,
        permissions: ["Read"],
        user: {
            ...minimalMemberResponse.user,
            id: "user_readonly_123456789012",
            handle: "viewer",
            name: "Read Only User",
        },
    },
    
    contributor: {
        ...minimalMemberResponse,
        id: "member_contributor_123456789012",
        isAdmin: false,
        permissions: ["Create", "Read", "Update"],
        user: {
            ...minimalMemberResponse.user,
            id: "user_contributor_123456789012",
            handle: "contributor",
            name: "Contributor User",
        },
    },
};

/**
 * Bot member examples (bots can be team members too)
 */
export const botMemberResponse = {
    __typename: "Member",
    id: "member_bot_123456789012345678",
    isAdmin: false,
    permissions: ["Read", "UseApi"],
    user: {
        __typename: "User",
        id: "user_bot_123456789012345678",
        handle: "testbot",
        name: "Test Bot",
        profileImage: null,
        isBot: true,
        updatedAt: "2024-01-15T15:00:00.000Z",
    },
    team: {
        __typename: "Team",
        id: "team_123456789012345678",
        handle: "testteam",
        name: "Test Team",
    },
    createdAt: "2024-01-15T15:00:00.000Z",
    updatedAt: "2024-01-15T15:00:00.000Z",
};