import type { TeamCreateInput, TeamUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for team-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Team creation form data
 */
export const minimalTeamCreateFormInput: Partial<TeamCreateInput> = {
    handle: "new-team",
    name: "New Team",
    isPrivate: false,
};

export const completeTeamCreateFormInput: Partial<TeamCreateInput> = {
    handle: "awesome-developers",
    isPrivate: false,
    isOpenToNewMembers: true,
    profileImage: null, // File object would go here
    bannerImage: null, // File object would go here
    tagsConnect: ["123456789012345678", "123456789012345679"], // Tag IDs
    translationsCreate: [
        {
            language: "en",
            name: "Awesome Developers",
            bio: "We build amazing AI-powered tools that help developers be more productive",
        },
    ],
    memberInvitesCreate: [
        { 
            userId: "123456789012345678",
            // Other MemberInviteCreateInput fields as needed
        },
    ],
};

/**
 * Team update form data
 */
export const minimalTeamUpdateFormInput = {
    name: "Updated Team Name",
    bio: "Updated team description",
};

export const completeTeamUpdateFormInput = {
    name: "Elite Developers",
    bio: "Leading team in AI development with 50+ successful projects delivered. Specializing in machine learning, automation, and developer tools.",
    isPrivate: false,
    profileImage: null, // File object for new image
    bannerImage: null, // File object for new image
    tags: ["ai", "ml", "automation", "enterprise"],
    links: [
        { type: "website", url: "https://elite-devs.com" },
        { type: "github", url: "https://github.com/elite-developers" },
        { type: "linkedin", url: "https://linkedin.com/company/elite-developers" },
        { type: "twitter", url: "https://twitter.com/elitedevs" },
    ],
};

/**
 * Team member invitation form data
 */
export const singleMemberInviteFormInput = {
    email: "newmember@example.com",
    role: "Member",
    message: "We'd love to have you join our team!",
};

export const bulkMemberInviteFormInput = {
    invitations: [
        {
            email: "developer1@example.com",
            role: "Developer",
            message: "Join us as a developer",
        },
        {
            email: "designer@example.com",
            role: "Designer",
            message: "We need your design skills",
        },
        {
            email: "manager@example.com",
            role: "Admin",
            message: "Help us manage the team",
        },
    ],
    sendEmails: true,
};

/**
 * Team role management form data
 */
export const createRoleFormInput = {
    name: "Developer",
    permissions: ["Read", "Write", "Comment"],
    description: "Team members who actively develop",
};

export const updateMemberRoleFormInput = {
    memberId: "member_123456789",
    roles: ["Developer", "Reviewer"],
    permissions: {
        canInvite: true,
        canRemove: false,
        canUpdateTeam: false,
        canManageRoles: false,
    },
};

/**
 * Team settings form data
 */
export const teamGeneralSettingsFormInput = {
    handle: "team-handle",
    name: "Team Name",
    bio: "Team description",
    isPrivate: false,
    requireApproval: true, // Require approval for new members
    allowGuestAccess: false,
};

export const teamNotificationSettingsFormInput = {
    newMemberNotifications: true,
    projectUpdateNotifications: true,
    mentionNotifications: true,
    weeklyDigest: true,
    notificationChannel: "email", // "email" | "slack" | "discord"
};

export const teamBillingSettingsFormInput = {
    billingEmail: "billing@team.com",
    plan: "premium", // "free" | "premium" | "enterprise"
    seats: 10,
    billingCycle: "monthly", // "monthly" | "yearly"
    paymentMethod: {
        type: "card",
        last4: "4242",
    },
};

/**
 * Team project form data
 */
export const teamProjectFormInput = {
    projectId: "project_123456789",
    teamRole: "owner", // "owner" | "contributor" | "viewer"
    permissions: {
        canEdit: true,
        canDelete: false,
        canTransfer: false,
        canInvite: true,
    },
};

/**
 * Leave team form data
 */
export const leaveTeamFormInput = {
    confirmText: "LEAVE",
    reason: "Moving to different project",
    feedback: "Great team, but I need to focus on other priorities",
    transferOwnership: null, // Member ID to transfer ownership to
};

/**
 * Form validation states
 */
export const teamFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            handle: "invalid handle!", // Contains spaces/special chars
            name: "", // Required but empty
        },
        errors: {
            handle: "Handle can only contain letters, numbers, and hyphens",
            name: "Team name is required",
        },
        touched: {
            handle: true,
            name: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalTeamCreateFormInput,
        errors: {},
        touched: {
            handle: true,
            name: true,
            isPrivate: true,
        },
        isValid: true,
        isSubmitting: false,
    },
};

/**
 * Helper function to create team form initial values
 */
export const createTeamFormInitialValues = (teamData?: Partial<any>) => ({
    handle: teamData?.handle || "",
    name: teamData?.name || "",
    bio: teamData?.bio || "",
    isPrivate: teamData?.isPrivate || false,
    tags: teamData?.tags || [],
    links: teamData?.links || [],
    ...teamData,
});

/**
 * Helper function to validate team handle
 */
export const validateTeamHandle = (handle: string): string | null => {
    if (!handle) return "Handle is required";
    if (handle.length < 3) return "Handle must be at least 3 characters";
    if (handle.length > 30) return "Handle must be less than 30 characters";
    if (!/^[a-zA-Z0-9-]+$/.test(handle)) {
        return "Handle can only contain letters, numbers, and hyphens";
    }
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformTeamFormToApiInput = (formData: any) => ({
    ...formData,
    // Convert file objects to upload format
    profileImage: formData.profileImage?.file || undefined,
    bannerImage: formData.bannerImage?.file || undefined,
    // Filter out empty links
    links: formData.links?.filter((link: any) => link.url) || [],
    // Convert tag strings to proper format
    tags: formData.tags?.map((tag: string) => ({ tag })) || [],
});

/**
 * Mock autocomplete suggestions
 */
export const mockTeamSuggestions = {
    handles: [
        "suggested-handle-1",
        "suggested-handle-2",
        "suggested-handle-3",
    ],
    tags: [
        { tag: "artificial-intelligence", count: 1234 },
        { tag: "web-development", count: 987 },
        { tag: "machine-learning", count: 654 },
        { tag: "automation", count: 543 },
    ],
    members: [
        { id: "user_123", handle: "johndoe", name: "John Doe" },
        { id: "user_456", handle: "janedoe", name: "Jane Doe" },
        { id: "user_789", handle: "bobsmith", name: "Bob Smith" },
    ],
};