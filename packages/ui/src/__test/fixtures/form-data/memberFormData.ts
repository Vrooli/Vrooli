import type { MemberUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for member-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Member update form data (members can only be updated, not created directly)
 */
export const minimalMemberUpdateFormInput: Partial<MemberUpdateInput> = {
    isAdmin: false,
    permissions: ["Read"],
};

export const completeMemberUpdateFormInput = {
    isAdmin: true,
    permissions: ["Create", "Read", "Update", "Delete", "UseApi", "Manage"],
    customRole: "Senior Developer",
    notes: "Core team member with full access to all features",
};

/**
 * Member role assignment form data
 */
export const assignBasicRoleFormInput = {
    permissions: ["Read", "Comment"],
    customRole: "Viewer",
};

export const assignDeveloperRoleFormInput = {
    permissions: ["Create", "Read", "Update", "UseApi"],
    customRole: "Developer",
    canInviteMembers: false,
    canManageRoles: false,
};

export const assignManagerRoleFormInput = {
    permissions: ["Create", "Read", "Update", "Delete", "Manage"],
    customRole: "Manager",
    canInviteMembers: true,
    canManageRoles: true,
    canRemoveMembers: true,
};

/**
 * Bulk member update form data
 */
export const bulkMemberUpdateFormInput = {
    memberIds: ["member_123456789", "member_987654321", "member_456789123"],
    permissions: ["Read", "Update"],
    isAdmin: false,
    applyToAll: true,
};

/**
 * Member permission sets (common combinations)
 */
export const memberPermissionSets = {
    readOnly: {
        permissions: ["Read"],
        isAdmin: false,
    },
    contributor: {
        permissions: ["Create", "Read", "Update"],
        isAdmin: false,
    },
    moderator: {
        permissions: ["Read", "Update", "Delete"],
        isAdmin: false,
    },
    developer: {
        permissions: ["Create", "Read", "Update", "UseApi"],
        isAdmin: false,
    },
    admin: {
        permissions: ["Create", "Read", "Update", "Delete", "UseApi", "Manage"],
        isAdmin: true,
    },
};

/**
 * Member removal form data
 */
export const removeMemberFormInput = {
    memberId: "member_123456789",
    reason: "Inactive for extended period",
    sendNotification: true,
    blockRejoin: false,
};

/**
 * Member transfer ownership form data
 */
export const transferOwnershipFormInput = {
    newOwnerId: "member_987654321",
    confirmText: "TRANSFER",
    reason: "Stepping down from leadership role",
    retainMembership: true,
    newRole: "Member",
};

/**
 * Form validation states
 */
export const memberFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            permissions: [], // Empty permissions not allowed
            isAdmin: true,
        },
        errors: {
            permissions: "At least one permission must be selected",
        },
        touched: {
            permissions: true,
            isAdmin: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalMemberUpdateFormInput,
        errors: {},
        touched: {
            isAdmin: true,
            permissions: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeMemberUpdateFormInput,
        errors: {},
        touched: {
            isAdmin: true,
            permissions: true,
        },
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create member form initial values
 */
export const createMemberFormInitialValues = (memberData?: Partial<any>) => ({
    isAdmin: memberData?.isAdmin || false,
    permissions: memberData?.permissions || ["Read"],
    customRole: memberData?.customRole || "",
    notes: memberData?.notes || "",
    ...memberData,
});

/**
 * Helper function to validate permissions
 */
export const validateMemberPermissions = (permissions: string[]): string | null => {
    if (!permissions || permissions.length === 0) {
        return "At least one permission must be selected";
    }
    
    const validPermissions = ["Create", "Read", "Update", "Delete", "UseApi", "Manage"];
    const invalidPermissions = permissions.filter(p => !validPermissions.includes(p));
    
    if (invalidPermissions.length > 0) {
        return `Invalid permissions: ${invalidPermissions.join(", ")}`;
    }
    
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformMemberFormToApiInput = (formData: any) => ({
    id: formData.id || formData.memberId, // Required for update
    isAdmin: formData.isAdmin || false,
    permissions: JSON.stringify(formData.permissions || []),
    // Note: customRole and notes would typically be stored elsewhere
});

/**
 * Mock member suggestions for autocomplete
 */
export const mockMemberSuggestions = {
    members: [
        { 
            id: "member_123456789", 
            user: { 
                handle: "johndoe", 
                name: "John Doe",
                profileImage: null,
            },
            isAdmin: false,
            permissions: ["Read", "Update"],
        },
        { 
            id: "member_987654321", 
            user: { 
                handle: "janedoe", 
                name: "Jane Doe",
                profileImage: null,
            },
            isAdmin: true,
            permissions: ["Create", "Read", "Update", "Delete", "UseApi", "Manage"],
        },
        { 
            id: "member_456789123", 
            user: { 
                handle: "bobsmith", 
                name: "Bob Smith",
                profileImage: null,
            },
            isAdmin: false,
            permissions: ["Read"],
        },
    ],
    roles: [
        { name: "Admin", permissions: ["Create", "Read", "Update", "Delete", "UseApi", "Manage"] },
        { name: "Developer", permissions: ["Create", "Read", "Update", "UseApi"] },
        { name: "Moderator", permissions: ["Read", "Update", "Delete"] },
        { name: "Contributor", permissions: ["Create", "Read", "Update"] },
        { name: "Viewer", permissions: ["Read"] },
    ],
};

/**
 * Permission descriptions for UI display
 */
export const permissionDescriptions = {
    Create: "Can create new content and resources",
    Read: "Can view content and resources",
    Update: "Can edit existing content and resources",
    Delete: "Can remove content and resources",
    UseApi: "Can access API endpoints and integrations",
    Manage: "Can manage team settings and members",
};