import type { MemberInviteCreateInput, MemberInviteUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for member invite-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Member invite creation form data
 */
export const minimalMemberInviteCreateFormInput: Partial<MemberInviteCreateInput> = {
    teamId: "123456789012345681",
    userId: "123456789012345683",
};

export const completeMemberInviteCreateFormInput = {
    teamId: "123456789012345681",
    userId: "123456789012345683",
    message: "You've been invited to join our team! We'd love to have you collaborate with us on exciting projects.",
    willBeAdmin: false,
    permissions: ["Read", "Create", "Update"],
};

/**
 * Bulk member invite creation form data
 */
export const bulkMemberInviteCreateFormInput = {
    teamId: "123456789012345681",
    invites: [
        {
            userId: "123456789012345683",
            message: "Welcome to the team! Looking forward to working with you.",
            willBeAdmin: false,
            permissions: ["Read", "Create", "Update"],
        },
        {
            userId: "123456789012345684",
            message: "Your expertise would be valuable to our team. Please join us!",
            willBeAdmin: false,
            permissions: ["Read", "Create", "Update", "Delete"],
        },
        {
            userId: "123456789012345685",
            message: "We'd like you to join as a team admin to help manage our projects.",
            willBeAdmin: true,
            permissions: ["Read", "Create", "Update", "Delete", "Manage", "UseApi"],
        },
    ],
};

/**
 * Member invite by email form data
 */
export const memberInviteByEmailFormInput = {
    teamId: "123456789012345681",
    emails: [
        "newmember@example.com",
        "developer@company.com",
        "designer@agency.com",
    ],
    message: "You're invited to join our team on Vrooli. Click the link below to accept.",
    willBeAdmin: false,
    permissions: ["Read", "Create"],
    sendEmail: true,
};

/**
 * Member invite with admin privileges form data
 */
export const memberInviteAdminFormInput = {
    teamId: "123456789012345681",
    userId: "123456789012345683",
    message: "You're invited to join our team as an administrator. You'll have full access to manage team resources.",
    willBeAdmin: true,
    permissions: ["Read", "Create", "Update", "Delete", "Manage", "UseApi"],
};

/**
 * Member invite with limited permissions form data
 */
export const memberInviteViewerFormInput = {
    teamId: "123456789012345681",
    userId: "123456789012345683",
    message: "You're invited to view our team's projects and resources.",
    willBeAdmin: false,
    permissions: ["Read"],
};

/**
 * Member invite update form data
 */
export const memberInviteUpdateFormInput = {
    message: "Updated invitation message with additional details about team expectations.",
    willBeAdmin: true,
    permissions: ["Read", "Create", "Update", "Delete"],
};

/**
 * Member invite response form data
 */
export const memberInviteAcceptFormInput = {
    inviteId: "123456789012345678",
    response: "accepted",
    message: "Thanks for the invite! Excited to join the team.",
};

export const memberInviteDeclineFormInput = {
    inviteId: "123456789012345678",
    response: "declined",
    message: "Thank you for the invitation, but I'm unable to join at this time.",
    reason: "time_constraints", // "time_constraints" | "not_interested" | "already_member" | "other"
};

/**
 * Member invite with custom role form data
 */
export const memberInviteWithRoleFormInput = {
    teamId: "123456789012345681",
    userId: "123456789012345683",
    message: "Join our team as a Senior Developer with specific project responsibilities.",
    willBeAdmin: false,
    permissions: ["Read", "Create", "Update", "UseApi"],
    customRole: "Senior Developer",
    roleDescription: "Lead developer responsible for API integrations and code reviews",
};

/**
 * Form validation states
 */
export const memberInviteFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            teamId: "", // Required but empty
            userId: "invalid-id", // Invalid format
            permissions: [], // Empty permissions
            message: "A".repeat(4097), // Too long (max 4096)
        },
        errors: {
            teamId: "Please select a team",
            userId: "Invalid user ID format",
            permissions: "At least one permission must be selected",
            message: "Message is too long (max 4096 characters)",
        },
        touched: {
            teamId: true,
            userId: true,
            permissions: true,
            message: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: completeMemberInviteCreateFormInput,
        errors: {},
        touched: {
            teamId: true,
            userId: true,
            message: true,
            willBeAdmin: true,
            permissions: true,
        },
        isValid: true,
        isSubmitting: false,
    },
};

/**
 * Helper function to create member invite form initial values
 */
export const createMemberInviteFormInitialValues = (inviteData?: Partial<any>) => ({
    teamId: inviteData?.teamId || "",
    userId: inviteData?.userId || "",
    message: inviteData?.message || "",
    willBeAdmin: inviteData?.willBeAdmin || false,
    permissions: inviteData?.permissions || ["Read"],
    ...inviteData,
});

/**
 * Helper function to validate permissions selection
 */
export const validateMemberInvitePermissions = (permissions: string[]): string | null => {
    if (!permissions || permissions.length === 0) {
        return "At least one permission must be selected";
    }
    
    const validPermissions = ["Create", "Read", "Update", "Delete", "UseApi", "Manage"];
    const invalidPermissions = permissions.filter(p => !validPermissions.includes(p));
    
    if (invalidPermissions.length > 0) {
        return `Invalid permissions: ${invalidPermissions.join(", ")}`;
    }
    
    // Admin should have all permissions
    if (permissions.includes("Manage") && permissions.length < validPermissions.length) {
        return "Admins should have all permissions";
    }
    
    return null;
};

/**
 * Helper function to validate invite message
 */
export const validateMemberInviteMessage = (message: string): string | null => {
    if (message && message.length > 4096) {
        return "Message is too long (max 4096 characters)";
    }
    return null;
};

/**
 * Helper function to validate user selection
 */
export const validateUserSelection = (userId: string): string | null => {
    if (!userId) return "Please select a user to invite";
    if (!/^\d{18,19}$/.test(userId)) return "Invalid user ID format";
    return null;
};

/**
 * Helper function to validate team selection
 */
export const validateTeamSelection = (teamId: string): string | null => {
    if (!teamId) return "Please select a team";
    if (!/^\d{18,19}$/.test(teamId)) return "Invalid team ID format";
    return null;
};

/**
 * Helper function to validate email list
 */
export const validateEmailList = (emails: string[]): string | null => {
    if (!emails || emails.length === 0) return "Please enter at least one email";
    
    const invalidEmails = emails.filter(email => 
        !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
    );
    
    if (invalidEmails.length > 0) {
        return `Invalid email format: ${invalidEmails.join(", ")}`;
    }
    
    if (emails.length > 50) return "Cannot invite more than 50 users at once";
    
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformMemberInviteFormToApiInput = (formData: any) => ({
    id: formData.id || formData.inviteId,
    message: formData.message?.trim() || undefined,
    willBeAdmin: formData.willBeAdmin || false,
    willHavePermissions: JSON.stringify(formData.permissions || []),
    teamConnect: formData.teamId,
    userConnect: formData.userId,
    // Handle bulk invites
    invites: formData.invites?.map((invite: any) => ({
        userConnect: invite.userId,
        message: invite.message?.trim() || undefined,
        willBeAdmin: invite.willBeAdmin || false,
        willHavePermissions: JSON.stringify(invite.permissions || []),
    })) || undefined,
    // Handle email invites
    emailInvites: formData.emails?.map((email: string) => ({
        email: email.trim(),
        sendEmail: formData.sendEmail || false,
    })) || undefined,
});

/**
 * Mock user suggestions for invite form
 */
export const mockInviteUserSuggestions = [
    { id: "123456789012345683", handle: "johndoe", name: "John Doe", avatar: null },
    { id: "123456789012345684", handle: "janedoe", name: "Jane Doe", avatar: null },
    { id: "123456789012345685", handle: "bobsmith", name: "Bob Smith", avatar: null },
    { id: "123456789012345686", handle: "alice", name: "Alice Johnson", avatar: null },
    { id: "123456789012345687", handle: "charlie", name: "Charlie Brown", avatar: null },
];

/**
 * Mock team suggestions for invite form
 */
export const mockInviteTeamSuggestions = [
    { id: "123456789012345681", handle: "awesome-developers", name: "Awesome Developers", memberCount: 12 },
    { id: "123456789012345682", handle: "design-team", name: "Design Team", memberCount: 8 },
    { id: "123456789012345688", handle: "marketing-squad", name: "Marketing Squad", memberCount: 15 },
    { id: "123456789012345689", handle: "research-lab", name: "Research Lab", memberCount: 6 },
];

/**
 * Mock invite templates
 */
export const mockMemberInviteTemplates = {
    standard: "You've been invited to join our team!",
    detailed: "We'd love to have you join our team. Your skills and experience would be a valuable addition to our projects.",
    admin: "You're invited to join our team as an administrator. You'll have full access to manage team resources and members.",
    developer: "Join our development team! We're working on exciting projects and could use your expertise.",
    viewer: "You're invited to view our team's projects and resources. This will give you read-only access to our work.",
    custom: "",
};

/**
 * Mock permission presets
 */
export const mockPermissionPresets = {
    viewer: {
        name: "Viewer",
        permissions: ["Read"],
        description: "Can view team resources",
    },
    contributor: {
        name: "Contributor",
        permissions: ["Read", "Create", "Update"],
        description: "Can create and edit content",
    },
    developer: {
        name: "Developer",
        permissions: ["Read", "Create", "Update", "UseApi"],
        description: "Can develop with API access",
    },
    moderator: {
        name: "Moderator",
        permissions: ["Read", "Update", "Delete"],
        description: "Can moderate content",
    },
    admin: {
        name: "Admin",
        permissions: ["Read", "Create", "Update", "Delete", "UseApi", "Manage"],
        description: "Full team management access",
    },
};

/**
 * Mock invite status states
 */
export const mockMemberInviteStates = {
    pending: {
        status: "pending",
        sentAt: new Date().toISOString(),
        expiresAt: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(), // 7 days
    },
    accepted: {
        status: "accepted",
        sentAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
        acceptedAt: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000).toISOString(),
        memberSince: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000).toISOString(),
    },
    declined: {
        status: "declined",
        sentAt: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString(),
        declinedAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
        reason: "Not interested in joining at this time",
    },
    expired: {
        status: "expired",
        sentAt: new Date(Date.now() - 10 * 24 * 60 * 60 * 1000).toISOString(),
        expiresAt: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString(),
    },
};

/**
 * Permission descriptions for UI display
 */
export const permissionDescriptions = {
    Create: "Can create new content and resources",
    Read: "Can view team content and resources",
    Update: "Can edit existing content and resources",
    Delete: "Can remove content and resources",
    UseApi: "Can access API endpoints and integrations",
    Manage: "Can manage team settings and members",
};