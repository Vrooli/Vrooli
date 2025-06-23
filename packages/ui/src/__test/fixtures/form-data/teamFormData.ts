/**
 * Team Form Data Fixtures
 * 
 * Provides comprehensive form data fixtures for team creation, editing,
 * and member management forms with React Hook Form integration.
 */

import { useForm, type UseFormReturn, type FieldErrors } from "react-hook-form";
import { yupResolver } from "@hookform/resolvers/yup";
import * as yup from "yup";
import { act, waitFor } from "@testing-library/react";
import type { 
    TeamCreateInput,
    TeamUpdateInput,
    Team,
    MemberRole,
} from "@vrooli/shared";
import { 
    teamValidation,
    MemberRole as MemberRoleEnum,
} from "@vrooli/shared";

/**
 * UI-specific form data for team creation
 */
export interface TeamCreateFormData {
    // Basic info
    handle: string;
    name: string;
    bio?: string;
    
    // Privacy settings
    isPrivate: boolean;
    
    // Resources
    resourceListCreate?: {
        name: string;
        description?: string;
    };
    
    // Members to invite (UI-specific)
    inviteMembers?: Array<{
        email?: string;
        handle?: string;
        role: MemberRole;
        message?: string;
    }>;
    
    // Settings
    isOpenToNewMembers?: boolean;
    requireInviteApproval?: boolean;
}

/**
 * UI-specific form data for team profile editing
 */
export interface TeamProfileFormData {
    handle: string;
    name: string;
    bio?: string;
    
    // Privacy settings
    isPrivate: boolean;
    isOpenToNewMembers?: boolean;
    requireInviteApproval?: boolean;
    
    // Permissions
    permissions?: {
        canMemebersInvite: boolean;
        canMembersEdit: boolean;
        canMembersDelete: boolean;
    };
}

/**
 * UI-specific form data for member management
 */
export interface TeamMemberFormData {
    memberIdOrHandle: string;
    role: MemberRole;
    permissions?: {
        canInvite: boolean;
        canEdit: boolean;
        canDelete: boolean;
        canManageRoles: boolean;
    };
}

/**
 * Extended form state with team-specific properties
 */
export interface TeamFormState<T = TeamCreateFormData> {
    values: T;
    errors: Record<string, string>;
    touched: Record<string, boolean>;
    isDirty: boolean;
    isSubmitting: boolean;
    isValid: boolean;
    isValidating?: boolean;
    
    // Team-specific state
    teamState?: {
        isCheckin

gHandleAvailability?: boolean;
        handleAvailable?: boolean;
        currentTeam?: Partial<Team>;
        memberCount?: number;
        pendingInvites?: number;
    };
    
    // Member management state
    memberState?: {
        members: Array<{
            id: string;
            handle: string;
            name: string;
            role: MemberRole;
        }>;
        isLoadingMembers: boolean;
    };
}

/**
 * Team form data factory with React Hook Form integration
 */
export class TeamFormDataFactory {
    /**
     * Create validation schema for team creation
     */
    private createTeamCreationSchema(): yup.ObjectSchema<TeamCreateFormData> {
        return yup.object({
            handle: yup
                .string()
                .required("Team handle is required")
                .min(3, "Handle must be at least 3 characters")
                .max(30, "Handle cannot exceed 30 characters")
                .matches(/^[a-zA-Z0-9_-]+$/, "Handle can only contain letters, numbers, underscores and hyphens"),
                
            name: yup
                .string()
                .required("Team name is required")
                .min(1, "Team name cannot be empty")
                .max(100, "Team name cannot exceed 100 characters"),
                
            bio: yup
                .string()
                .max(1000, "Bio cannot exceed 1000 characters")
                .optional(),
                
            isPrivate: yup
                .boolean()
                .required("Privacy setting is required"),
                
            resourceListCreate: yup.object({
                name: yup.string().required("Resource list name is required"),
                description: yup.string().optional(),
            }).optional(),
            
            inviteMembers: yup.array(
                yup.object({
                    email: yup.string().email("Invalid email address").optional(),
                    handle: yup.string().optional(),
                    role: yup
                        .mixed<MemberRole>()
                        .oneOf(Object.values(MemberRoleEnum))
                        .required("Member role is required"),
                    message: yup.string().max(500, "Invitation message is too long").optional(),
                }).test("email-or-handle", "Either email or handle is required", function(value) {
                    return !!(value?.email || value?.handle);
                }),
            ).optional(),
            
            isOpenToNewMembers: yup.boolean().optional(),
            requireInviteApproval: yup.boolean().optional(),
        }).defined();
    }
    
    /**
     * Create validation schema for team profile
     */
    private createTeamProfileSchema(): yup.ObjectSchema<TeamProfileFormData> {
        return yup.object({
            handle: yup
                .string()
                .required("Team handle is required")
                .min(3, "Handle must be at least 3 characters")
                .max(30, "Handle cannot exceed 30 characters")
                .matches(/^[a-zA-Z0-9_-]+$/, "Handle can only contain letters, numbers, underscores and hyphens"),
                
            name: yup
                .string()
                .required("Team name is required")
                .min(1, "Team name cannot be empty")
                .max(100, "Team name cannot exceed 100 characters"),
                
            bio: yup
                .string()
                .max(1000, "Bio cannot exceed 1000 characters")
                .optional(),
                
            isPrivate: yup
                .boolean()
                .required("Privacy setting is required"),
                
            isOpenToNewMembers: yup.boolean().optional(),
            requireInviteApproval: yup.boolean().optional(),
            
            permissions: yup.object({
                canMemebersInvite: yup.boolean().required(),
                canMembersEdit: yup.boolean().required(),
                canMembersDelete: yup.boolean().required(),
            }).optional(),
        }).defined();
    }
    
    /**
     * Create validation schema for member management
     */
    private createMemberManagementSchema(): yup.ObjectSchema<TeamMemberFormData> {
        return yup.object({
            memberIdOrHandle: yup
                .string()
                .required("Member selection is required"),
                
            role: yup
                .mixed<MemberRole>()
                .oneOf(Object.values(MemberRoleEnum))
                .required("Member role is required"),
                
            permissions: yup.object({
                canInvite: yup.boolean().required(),
                canEdit: yup.boolean().required(),
                canDelete: yup.boolean().required(),
                canManageRoles: yup.boolean().required(),
            }).optional(),
        }).defined();
    }
    
    /**
     * Generate unique ID for testing
     */
    private generateId(): string {
        return `team_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create team creation form data for different scenarios
     */
    createTeamFormData(
        scenario: "empty" | "minimal" | "complete" | "invalid" | "privateTeam" | "withInvites" | "openTeam" | "partiallyCompleted",
    ): TeamCreateFormData {
        switch (scenario) {
            case "empty":
                return {
                    handle: "",
                    name: "",
                    isPrivate: false,
                };
                
            case "minimal":
                return {
                    handle: "myteam",
                    name: "My Team",
                    isPrivate: false,
                };
                
            case "complete":
                return {
                    handle: "dev-team",
                    name: "Development Team",
                    bio: "A team of passionate developers building amazing software",
                    isPrivate: false,
                    resourceListCreate: {
                        name: "Team Resources",
                        description: "Shared resources for the team",
                    },
                    inviteMembers: [
                        {
                            email: "alice@example.com",
                            role: MemberRoleEnum.Member,
                            message: "Welcome to the team!",
                        },
                        {
                            handle: "bob",
                            role: MemberRoleEnum.Admin,
                            message: "Join us as an admin",
                        },
                    ],
                    isOpenToNewMembers: true,
                    requireInviteApproval: false,
                };
                
            case "invalid":
                return {
                    handle: "a", // Too short
                    name: "", // Empty
                    isPrivate: false,
                    inviteMembers: [
                        {
                            // Missing both email and handle
                            role: MemberRoleEnum.Member,
                        },
                    ],
                };
                
            case "privateTeam":
                return {
                    handle: "secret-team",
                    name: "Secret Team",
                    bio: "Private team for special projects",
                    isPrivate: true,
                    isOpenToNewMembers: false,
                    requireInviteApproval: true,
                };
                
            case "withInvites":
                return {
                    handle: "collaborative-team",
                    name: "Collaborative Team",
                    isPrivate: false,
                    inviteMembers: [
                        {
                            email: "member1@example.com",
                            role: MemberRoleEnum.Member,
                        },
                        {
                            email: "member2@example.com",
                            role: MemberRoleEnum.Member,
                        },
                        {
                            email: "admin@example.com",
                            role: MemberRoleEnum.Admin,
                        },
                    ],
                };
                
            case "openTeam":
                return {
                    handle: "open-source-team",
                    name: "Open Source Contributors",
                    bio: "A team open to all contributors",
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    requireInviteApproval: false,
                };
                
            case "partiallyCompleted":
                return {
                    handle: "partial-team",
                    name: "Partial Team",
                    isPrivate: false,
                    inviteMembers: [
                        {
                            email: "alice@example.com",
                            // Missing role
                            role: "" as any,
                        },
                    ],
                };
                
            default:
                throw new Error(`Unknown team form scenario: ${scenario}`);
        }
    }
    
    /**
     * Create team profile form data for different scenarios
     */
    createProfileFormData(
        scenario: "minimal" | "complete" | "invalid" | "restrictive" | "permissive",
    ): TeamProfileFormData {
        switch (scenario) {
            case "minimal":
                return {
                    handle: "myteam",
                    name: "My Team",
                    isPrivate: false,
                };
                
            case "complete":
                return {
                    handle: "established-team",
                    name: "Established Team",
                    bio: "A well-established team with clear goals and processes",
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    requireInviteApproval: true,
                    permissions: {
                        canMemebersInvite: true,
                        canMembersEdit: true,
                        canMembersDelete: false,
                    },
                };
                
            case "invalid":
                return {
                    handle: "", // Empty
                    name: "", // Empty
                    isPrivate: false,
                };
                
            case "restrictive":
                return {
                    handle: "restricted-team",
                    name: "Restricted Access Team",
                    bio: "Highly controlled team with limited permissions",
                    isPrivate: true,
                    isOpenToNewMembers: false,
                    requireInviteApproval: true,
                    permissions: {
                        canMemebersInvite: false,
                        canMembersEdit: false,
                        canMembersDelete: false,
                    },
                };
                
            case "permissive":
                return {
                    handle: "open-team",
                    name: "Open Collaboration Team",
                    bio: "Everyone can contribute",
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    requireInviteApproval: false,
                    permissions: {
                        canMemebersInvite: true,
                        canMembersEdit: true,
                        canMembersDelete: true,
                    },
                };
                
            default:
                throw new Error(`Unknown profile scenario: ${scenario}`);
        }
    }
    
    /**
     * Create member management form data
     */
    createMemberFormData(
        scenario: "changeRole" | "grantAdmin" | "removeAdmin" | "restrictPermissions",
    ): TeamMemberFormData {
        switch (scenario) {
            case "changeRole":
                return {
                    memberIdOrHandle: "member123",
                    role: MemberRoleEnum.Member,
                    permissions: {
                        canInvite: true,
                        canEdit: false,
                        canDelete: false,
                        canManageRoles: false,
                    },
                };
                
            case "grantAdmin":
                return {
                    memberIdOrHandle: "member456",
                    role: MemberRoleEnum.Admin,
                    permissions: {
                        canInvite: true,
                        canEdit: true,
                        canDelete: true,
                        canManageRoles: true,
                    },
                };
                
            case "removeAdmin":
                return {
                    memberIdOrHandle: "admin789",
                    role: MemberRoleEnum.Member,
                    permissions: {
                        canInvite: true,
                        canEdit: true,
                        canDelete: false,
                        canManageRoles: false,
                    },
                };
                
            case "restrictPermissions":
                return {
                    memberIdOrHandle: "member999",
                    role: MemberRoleEnum.Member,
                    permissions: {
                        canInvite: false,
                        canEdit: false,
                        canDelete: false,
                        canManageRoles: false,
                    },
                };
                
            default:
                throw new Error(`Unknown member scenario: ${scenario}`);
        }
    }
    
    /**
     * Create form state for different scenarios
     */
    createFormState<T extends TeamCreateFormData | TeamProfileFormData | TeamMemberFormData>(
        scenario: "pristine" | "dirty" | "submitting" | "withErrors" | "valid" | "checkingHandle",
        formType: "create" | "profile" | "member" = "create",
    ): TeamFormState<T> {
        let baseFormData: any;
        
        switch (formType) {
            case "create":
                baseFormData = this.createTeamFormData("complete");
                break;
            case "profile":
                baseFormData = this.createProfileFormData("complete");
                break;
            case "member":
                baseFormData = this.createMemberFormData("changeRole");
                break;
        }
        
        switch (scenario) {
            case "pristine":
                return {
                    values: (formType === "create" 
                        ? this.createTeamFormData("empty")
                        : formType === "profile"
                        ? this.createProfileFormData("minimal")
                        : this.createMemberFormData("changeRole")) as T,
                    errors: {},
                    touched: {},
                    isDirty: false,
                    isSubmitting: false,
                    isValid: false,
                };
                
            case "dirty":
                return {
                    values: baseFormData as T,
                    errors: {},
                    touched: { handle: true, name: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                };
                
            case "submitting":
                return {
                    values: baseFormData as T,
                    errors: {},
                    touched: Object.keys(baseFormData).reduce((acc, key) => ({
                        ...acc,
                        [key]: true,
                    }), {}),
                    isDirty: true,
                    isSubmitting: true,
                    isValid: true,
                };
                
            case "withErrors":
                const errors = formType === "create" 
                    ? {
                        handle: "This handle is already taken",
                        name: "Team name is required",
                    }
                    : formType === "profile"
                    ? {
                        handle: "Invalid handle format",
                        bio: "Bio is too long",
                    }
                    : {
                        memberIdOrHandle: "Member not found",
                        role: "Invalid role",
                    };
                    
                return {
                    values: (formType === "create"
                        ? this.createTeamFormData("invalid")
                        : formType === "profile"
                        ? this.createProfileFormData("invalid")
                        : this.createMemberFormData("changeRole")) as T,
                    errors,
                    touched: Object.keys(errors).reduce((acc, key) => ({
                        ...acc,
                        [key]: true,
                    }), {}),
                    isDirty: true,
                    isSubmitting: false,
                    isValid: false,
                };
                
            case "valid":
                return {
                    values: baseFormData as T,
                    errors: {},
                    touched: Object.keys(baseFormData).reduce((acc, key) => ({
                        ...acc,
                        [key]: true,
                    }), {}),
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                };
                
            case "checkingHandle":
                return {
                    values: baseFormData as T,
                    errors: {},
                    touched: { handle: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: false,
                    isValidating: true,
                    teamState: {
                        isCheckingHandleAvailability: true,
                    },
                };
                
            default:
                throw new Error(`Unknown form state scenario: ${scenario}`);
        }
    }
    
    /**
     * Create React Hook Form instance
     */
    createFormInstance<T extends TeamCreateFormData | TeamProfileFormData | TeamMemberFormData>(
        formType: "create" | "profile" | "member",
        initialData?: Partial<T>,
    ): UseFormReturn<T> {
        let defaultValues: any;
        let resolver: any;
        
        switch (formType) {
            case "create":
                defaultValues = {
                    ...this.createTeamFormData("empty"),
                    ...initialData,
                };
                resolver = yupResolver(this.createTeamCreationSchema());
                break;
                
            case "profile":
                defaultValues = {
                    ...this.createProfileFormData("minimal"),
                    ...initialData,
                };
                resolver = yupResolver(this.createTeamProfileSchema());
                break;
                
            case "member":
                defaultValues = {
                    ...this.createMemberFormData("changeRole"),
                    ...initialData,
                };
                resolver = yupResolver(this.createMemberManagementSchema());
                break;
        }
        
        return useForm<T>({
            mode: "onChange",
            reValidateMode: "onChange",
            shouldFocusError: true,
            defaultValues,
            resolver,
        });
    }
    
    /**
     * Validate form data using real validation
     */
    async validateFormData<T extends TeamCreateFormData | TeamProfileFormData>(
        formData: T,
        formType: "create" | "profile",
    ): Promise<{
        isValid: boolean;
        errors?: Record<string, string>;
        apiInput?: TeamCreateInput | TeamUpdateInput;
    }> {
        try {
            const schema = formType === "create" 
                ? this.createTeamCreationSchema()
                : this.createTeamProfileSchema();
                
            await schema.validate(formData, { abortEarly: false });
            
            const apiInput = formType === "create"
                ? this.transformCreateToAPIInput(formData as TeamCreateFormData)
                : this.transformProfileToAPIInput(formData as TeamProfileFormData);
                
            if (formType === "create") {
                await teamValidation.create.validate(apiInput);
            } else {
                await teamValidation.update.validate(apiInput);
            }
            
            return {
                isValid: true,
                apiInput,
            };
        } catch (error: any) {
            const errors: Record<string, string> = {};
            
            if (error.inner) {
                error.inner.forEach((err: any) => {
                    if (err.path) {
                        errors[err.path] = err.message;
                    }
                });
            } else if (error.message) {
                errors.general = error.message;
            }
            
            return {
                isValid: false,
                errors,
            };
        }
    }
    
    /**
     * Transform create form data to API input
     */
    private transformCreateToAPIInput(formData: TeamCreateFormData): TeamCreateInput {
        const input: TeamCreateInput = {
            id: this.generateId(),
            handle: formData.handle,
            translationsCreate: [{
                id: this.generateId(),
                language: "en",
                name: formData.name,
                bio: formData.bio,
            }],
            isPrivate: formData.isPrivate,
            isOpenToNewMembers: formData.isOpenToNewMembers,
        };
        
        if (formData.resourceListCreate) {
            input.resourceListsCreate = [{
                id: this.generateId(),
                translationsCreate: [{
                    id: this.generateId(),
                    language: "en",
                    name: formData.resourceListCreate.name,
                    description: formData.resourceListCreate.description,
                }],
            }];
        }
        
        return input;
    }
    
    /**
     * Transform profile form data to API input
     */
    private transformProfileToAPIInput(formData: TeamProfileFormData): TeamUpdateInput {
        return {
            id: this.generateId(),
            handle: formData.handle,
            translationsUpdate: [{
                id: this.generateId(),
                language: "en",
                name: formData.name,
                bio: formData.bio,
            }],
            isPrivate: formData.isPrivate,
            isOpenToNewMembers: formData.isOpenToNewMembers,
        };
    }
}

/**
 * Form interaction simulator for team forms
 */
export class TeamFormInteractionSimulator {
    private interactionDelay = 100;
    
    constructor(delay?: number) {
        this.interactionDelay = delay || 100;
    }
    
    /**
     * Simulate team creation flow
     */
    async simulateTeamCreationFlow(
        formInstance: UseFormReturn<TeamCreateFormData>,
        formData: TeamCreateFormData,
    ): Promise<void> {
        // Type handle with availability check
        for (let i = 1; i <= formData.handle.length; i++) {
            const partialHandle = formData.handle.substring(0, i);
            await this.fillField(formInstance, "handle", partialHandle);
            
            // Simulate availability check after 3 characters
            if (i >= 3) {
                await new Promise(resolve => setTimeout(resolve, 300));
            }
        }
        
        // Type team name
        await this.simulateTyping(formInstance, "name", formData.name);
        
        // Type bio if present
        if (formData.bio) {
            await this.simulateTyping(formInstance, "bio", formData.bio);
        }
        
        // Set privacy
        await this.fillField(formInstance, "isPrivate", formData.isPrivate);
        
        // Add member invites
        if (formData.inviteMembers) {
            for (let i = 0; i < formData.inviteMembers.length; i++) {
                const member = formData.inviteMembers[i];
                
                if (member.email) {
                    await this.simulateTyping(
                        formInstance, 
                        `inviteMembers.${i}.email` as any, 
                        member.email,
                    );
                } else if (member.handle) {
                    await this.simulateTyping(
                        formInstance, 
                        `inviteMembers.${i}.handle` as any, 
                        member.handle,
                    );
                }
                
                await this.fillField(
                    formInstance, 
                    `inviteMembers.${i}.role` as any, 
                    member.role,
                );
                
                if (member.message) {
                    await this.simulateTyping(
                        formInstance, 
                        `inviteMembers.${i}.message` as any, 
                        member.message,
                    );
                }
                
                await new Promise(resolve => setTimeout(resolve, 200));
            }
        }
        
        // Set team settings
        if (formData.isOpenToNewMembers !== undefined) {
            await this.fillField(formInstance, "isOpenToNewMembers", formData.isOpenToNewMembers);
        }
        
        if (formData.requireInviteApproval !== undefined) {
            await this.fillField(formInstance, "requireInviteApproval", formData.requireInviteApproval);
        }
    }
    
    /**
     * Simulate typing
     */
    private async simulateTyping(
        formInstance: UseFormReturn<any>,
        fieldName: string,
        text: string,
    ): Promise<void> {
        for (let i = 1; i <= text.length; i++) {
            await this.fillField(formInstance, fieldName, text.substring(0, i));
            await new Promise(resolve => setTimeout(resolve, 50));
        }
    }
    
    /**
     * Fill field helper
     */
    private async fillField(
        formInstance: UseFormReturn<any>,
        fieldName: string,
        value: any,
    ): Promise<void> {
        act(() => {
            formInstance.setValue(fieldName, value, {
                shouldDirty: true,
                shouldTouch: true,
                shouldValidate: true,
            });
        });
        
        await waitFor(() => {
            expect(formInstance.formState.isValidating).toBe(false);
        });
    }
}

// Export factory instances
export const teamFormFactory = new TeamFormDataFactory();
export const teamFormSimulator = new TeamFormInteractionSimulator();

// Export pre-configured scenarios
export const teamFormScenarios = {
    // Creation scenarios
    emptyTeamCreation: () => teamFormFactory.createFormState("pristine", "create"),
    validTeamCreation: () => teamFormFactory.createFormState("valid", "create"),
    teamCreationWithErrors: () => teamFormFactory.createFormState("withErrors", "create"),
    submittingTeamCreation: () => teamFormFactory.createFormState("submitting", "create"),
    
    // Team types
    minimalTeam: () => teamFormFactory.createTeamFormData("minimal"),
    completeTeam: () => teamFormFactory.createTeamFormData("complete"),
    privateTeam: () => teamFormFactory.createTeamFormData("privateTeam"),
    openTeam: () => teamFormFactory.createTeamFormData("openTeam"),
    teamWithInvites: () => teamFormFactory.createTeamFormData("withInvites"),
    
    // Profile scenarios
    teamProfile: () => teamFormFactory.createProfileFormData("complete"),
    restrictiveTeam: () => teamFormFactory.createProfileFormData("restrictive"),
    permissiveTeam: () => teamFormFactory.createProfileFormData("permissive"),
    
    // Member management
    changeMemberRole: () => teamFormFactory.createMemberFormData("changeRole"),
    grantAdminRole: () => teamFormFactory.createMemberFormData("grantAdmin"),
    removeAdminRole: () => teamFormFactory.createMemberFormData("removeAdmin"),
    
    // Interactive workflows
    async completeTeamCreationWorkflow(formInstance: UseFormReturn<TeamCreateFormData>) {
        const simulator = new TeamFormInteractionSimulator();
        const formData = teamFormFactory.createTeamFormData("complete");
        await simulator.simulateTeamCreationFlow(formInstance, formData);
        return formData;
    },
    
    async quickTeamSetup(formInstance: UseFormReturn<TeamCreateFormData>) {
        const simulator = new TeamFormInteractionSimulator(25); // Faster
        const formData = teamFormFactory.createTeamFormData("minimal");
        await simulator.simulateTeamCreationFlow(formInstance, formData);
        return formData;
    },
};
