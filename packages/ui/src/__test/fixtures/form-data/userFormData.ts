/**
 * User Form Data Fixtures
 * 
 * Provides comprehensive form data fixtures for user registration, profile editing,
 * and authentication forms with React Hook Form integration.
 */

import { useForm, type UseFormReturn, type FieldErrors } from "react-hook-form";
import { yupResolver } from "@hookform/resolvers/yup";
import * as yup from "yup";
import { act, waitFor } from "@testing-library/react";
import type { 
    UserCreateInput,
    UserUpdateInput,
    User,
    AccountStatus,
} from "@vrooli/shared";
import { 
    userValidation,
    AccountStatus as AccountStatusEnum,
    emailSignUpFormValidation,
} from "@vrooli/shared";

/**
 * UI-specific form data for user registration
 */
export interface UserRegistrationFormData {
    // Basic info
    email: string;
    password: string;
    confirmPassword: string;
    handle: string;
    name: string;
    
    // Terms & conditions
    agreeToTerms: boolean;
    agreeToPrivacyPolicy: boolean;
    
    // Optional profile
    bio?: string;
    theme?: string;
    
    // Marketing preferences
    marketingEmails?: boolean;
}

/**
 * UI-specific form data for user profile editing
 */
export interface UserProfileFormData {
    handle: string;
    name: string;
    bio?: string;
    theme?: string;
    language?: string;
    timezone?: string;
    
    // Privacy settings
    isPrivate?: boolean;
    isPrivateApis?: boolean;
    isPrivateProjects?: boolean;
    isPrivateRoutines?: boolean;
    isPrivatePullRequests?: boolean;
    isPrivateTeams?: boolean;
    
    // Notification preferences
    notificationSettings?: {
        emailsEnabled: boolean;
        pushEnabled: boolean;
        weeklyReports: boolean;
    };
}

/**
 * UI-specific form data for login
 */
export interface UserLoginFormData {
    emailOrHandle: string;
    password: string;
    rememberMe?: boolean;
}

/**
 * Extended form state with UI-specific properties
 */
export interface UserFormState<T = UserRegistrationFormData> {
    values: T;
    errors: Record<string, string>;
    touched: Record<string, boolean>;
    isDirty: boolean;
    isSubmitting: boolean;
    isValid: boolean;
    isValidating?: boolean;
    
    // Authentication state
    authState?: {
        isAuthenticated: boolean;
        isLoading: boolean;
        error?: string;
        user?: Partial<User>;
    };
    
    // Password strength
    passwordStrength?: {
        score: number; // 0-4
        feedback: string[];
        isStrong: boolean;
    };
}

/**
 * User form data factory with React Hook Form integration
 */
export class UserFormDataFactory {
    /**
     * Create validation schema for registration
     */
    private createRegistrationSchema(): yup.ObjectSchema<UserRegistrationFormData> {
        return yup.object({
            email: yup
                .string()
                .email("Please enter a valid email address")
                .required("Email is required")
                .max(255, "Email is too long"),
                
            password: yup
                .string()
                .required("Password is required")
                .min(8, "Password must be at least 8 characters")
                .matches(
                    /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]/,
                    "Password must contain uppercase, lowercase, number and special character",
                ),
                
            confirmPassword: yup
                .string()
                .required("Please confirm your password")
                .oneOf([yup.ref("password")], "Passwords must match"),
                
            handle: yup
                .string()
                .required("Username is required")
                .min(3, "Username must be at least 3 characters")
                .max(20, "Username cannot exceed 20 characters")
                .matches(/^[a-zA-Z0-9_]+$/, "Username can only contain letters, numbers and underscores"),
                
            name: yup
                .string()
                .required("Display name is required")
                .min(1, "Display name cannot be empty")
                .max(50, "Display name cannot exceed 50 characters"),
                
            agreeToTerms: yup
                .boolean()
                .oneOf([true], "You must accept the terms and conditions")
                .required("You must accept the terms and conditions"),
                
            agreeToPrivacyPolicy: yup
                .boolean()
                .oneOf([true], "You must accept the privacy policy")
                .required("You must accept the privacy policy"),
                
            bio: yup
                .string()
                .max(500, "Bio cannot exceed 500 characters")
                .optional(),
                
            theme: yup
                .string()
                .optional(),
                
            marketingEmails: yup
                .boolean()
                .optional(),
        }).defined();
    }
    
    /**
     * Create validation schema for profile editing
     */
    private createProfileSchema(): yup.ObjectSchema<UserProfileFormData> {
        return yup.object({
            handle: yup
                .string()
                .required("Username is required")
                .min(3, "Username must be at least 3 characters")
                .max(20, "Username cannot exceed 20 characters")
                .matches(/^[a-zA-Z0-9_]+$/, "Username can only contain letters, numbers and underscores"),
                
            name: yup
                .string()
                .required("Display name is required")
                .min(1, "Display name cannot be empty")
                .max(50, "Display name cannot exceed 50 characters"),
                
            bio: yup
                .string()
                .max(500, "Bio cannot exceed 500 characters")
                .optional(),
                
            theme: yup
                .string()
                .optional(),
                
            language: yup
                .string()
                .optional(),
                
            timezone: yup
                .string()
                .optional(),
                
            isPrivate: yup.boolean().optional(),
            isPrivateApis: yup.boolean().optional(),
            isPrivateProjects: yup.boolean().optional(),
            isPrivateRoutines: yup.boolean().optional(),
            isPrivatePullRequests: yup.boolean().optional(),
            isPrivateTeams: yup.boolean().optional(),
            
            notificationSettings: yup.object({
                emailsEnabled: yup.boolean().required(),
                pushEnabled: yup.boolean().required(),
                weeklyReports: yup.boolean().required(),
            }).optional(),
        }).defined();
    }
    
    /**
     * Create validation schema for login
     */
    private createLoginSchema(): yup.ObjectSchema<UserLoginFormData> {
        return yup.object({
            emailOrHandle: yup
                .string()
                .required("Email or username is required")
                .min(1, "Email or username cannot be empty"),
                
            password: yup
                .string()
                .required("Password is required")
                .min(1, "Password cannot be empty"),
                
            rememberMe: yup
                .boolean()
                .optional(),
        }).defined();
    }
    
    /**
     * Generate unique ID for testing
     */
    private generateId(): string {
        return `user_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create registration form data for different scenarios
     */
    createRegistrationFormData(
        scenario: "empty" | "minimal" | "complete" | "invalid" | "weakPassword" | "passwordMismatch" | "invalidEmail" | "partiallyCompleted",
    ): UserRegistrationFormData {
        switch (scenario) {
            case "empty":
                return {
                    email: "",
                    password: "",
                    confirmPassword: "",
                    handle: "",
                    name: "",
                    agreeToTerms: false,
                    agreeToPrivacyPolicy: false,
                };
                
            case "minimal":
                return {
                    email: "test@example.com",
                    password: "SecurePass123!",
                    confirmPassword: "SecurePass123!",
                    handle: "testuser",
                    name: "Test User",
                    agreeToTerms: true,
                    agreeToPrivacyPolicy: true,
                };
                
            case "complete":
                return {
                    email: "john.doe@example.com",
                    password: "SuperSecure123!",
                    confirmPassword: "SuperSecure123!",
                    handle: "johndoe",
                    name: "John Doe",
                    agreeToTerms: true,
                    agreeToPrivacyPolicy: true,
                    bio: "Software developer passionate about open source",
                    theme: "dark",
                    marketingEmails: true,
                };
                
            case "invalid":
                return {
                    email: "invalid-email",
                    password: "weak",
                    confirmPassword: "different",
                    handle: "a", // Too short
                    name: "",
                    agreeToTerms: false,
                    agreeToPrivacyPolicy: false,
                };
                
            case "weakPassword":
                return {
                    email: "user@example.com",
                    password: "password", // No uppercase, numbers, or special chars
                    confirmPassword: "password",
                    handle: "weakpassuser",
                    name: "Weak Pass User",
                    agreeToTerms: true,
                    agreeToPrivacyPolicy: true,
                };
                
            case "passwordMismatch":
                return {
                    email: "user@example.com",
                    password: "SecurePass123!",
                    confirmPassword: "DifferentPass123!",
                    handle: "mismatchuser",
                    name: "Mismatch User",
                    agreeToTerms: true,
                    agreeToPrivacyPolicy: true,
                };
                
            case "invalidEmail":
                return {
                    email: "not.an.email",
                    password: "SecurePass123!",
                    confirmPassword: "SecurePass123!",
                    handle: "invalidemailuser",
                    name: "Invalid Email User",
                    agreeToTerms: true,
                    agreeToPrivacyPolicy: true,
                };
                
            case "partiallyCompleted":
                return {
                    email: "partial@example.com",
                    password: "SecurePass123!",
                    confirmPassword: "SecurePass123!",
                    handle: "partialuser",
                    name: "Partial User",
                    agreeToTerms: true,
                    agreeToPrivacyPolicy: false, // Missing this
                };
                
            default:
                throw new Error(`Unknown registration scenario: ${scenario}`);
        }
    }
    
    /**
     * Create profile form data for different scenarios
     */
    createProfileFormData(
        scenario: "minimal" | "complete" | "invalid" | "privateProfile" | "notificationsOff",
    ): UserProfileFormData {
        switch (scenario) {
            case "minimal":
                return {
                    handle: "johndoe",
                    name: "John Doe",
                };
                
            case "complete":
                return {
                    handle: "johndoe",
                    name: "John Doe",
                    bio: "Full-stack developer with 10 years of experience",
                    theme: "dark",
                    language: "en",
                    timezone: "America/New_York",
                    isPrivate: false,
                    isPrivateApis: false,
                    isPrivateProjects: false,
                    isPrivateRoutines: false,
                    isPrivatePullRequests: false,
                    isPrivateTeams: false,
                    notificationSettings: {
                        emailsEnabled: true,
                        pushEnabled: true,
                        weeklyReports: true,
                    },
                };
                
            case "invalid":
                return {
                    handle: "a", // Too short
                    name: "", // Empty
                };
                
            case "privateProfile":
                return {
                    handle: "privateuser",
                    name: "Private User",
                    bio: "I value my privacy",
                    isPrivate: true,
                    isPrivateApis: true,
                    isPrivateProjects: true,
                    isPrivateRoutines: true,
                    isPrivatePullRequests: true,
                    isPrivateTeams: true,
                };
                
            case "notificationsOff":
                return {
                    handle: "quietuser",
                    name: "Quiet User",
                    notificationSettings: {
                        emailsEnabled: false,
                        pushEnabled: false,
                        weeklyReports: false,
                    },
                };
                
            default:
                throw new Error(`Unknown profile scenario: ${scenario}`);
        }
    }
    
    /**
     * Create login form data for different scenarios
     */
    createLoginFormData(
        scenario: "empty" | "withEmail" | "withHandle" | "invalid" | "rememberMe",
    ): UserLoginFormData {
        switch (scenario) {
            case "empty":
                return {
                    emailOrHandle: "",
                    password: "",
                };
                
            case "withEmail":
                return {
                    emailOrHandle: "user@example.com",
                    password: "SecurePass123!",
                };
                
            case "withHandle":
                return {
                    emailOrHandle: "johndoe",
                    password: "SecurePass123!",
                };
                
            case "invalid":
                return {
                    emailOrHandle: "",
                    password: "",
                };
                
            case "rememberMe":
                return {
                    emailOrHandle: "user@example.com",
                    password: "SecurePass123!",
                    rememberMe: true,
                };
                
            default:
                throw new Error(`Unknown login scenario: ${scenario}`);
        }
    }
    
    /**
     * Create form state for different scenarios
     */
    createFormState<T extends UserRegistrationFormData | UserProfileFormData | UserLoginFormData>(
        scenario: "pristine" | "dirty" | "submitting" | "withErrors" | "valid" | "authenticated",
        formType: "registration" | "profile" | "login" = "registration",
    ): UserFormState<T> {
        let baseFormData: any;
        
        switch (formType) {
            case "registration":
                baseFormData = this.createRegistrationFormData("complete");
                break;
            case "profile":
                baseFormData = this.createProfileFormData("complete");
                break;
            case "login":
                baseFormData = this.createLoginFormData("withEmail");
                break;
        }
        
        switch (scenario) {
            case "pristine":
                return {
                    values: (formType === "registration" 
                        ? this.createRegistrationFormData("empty")
                        : formType === "profile"
                        ? this.createProfileFormData("minimal")
                        : this.createLoginFormData("empty")) as T,
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
                    touched: { email: true, handle: true, name: true },
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
                const errors = formType === "registration" 
                    ? {
                        email: "Email is already taken",
                        handle: "Username is already taken",
                        password: "Password is too weak",
                    }
                    : formType === "profile"
                    ? {
                        handle: "Username is already taken",
                        name: "Display name is required",
                    }
                    : {
                        emailOrHandle: "Invalid email or username",
                        password: "Incorrect password",
                    };
                    
                return {
                    values: (formType === "registration"
                        ? this.createRegistrationFormData("invalid")
                        : formType === "profile"
                        ? this.createProfileFormData("invalid")
                        : this.createLoginFormData("invalid")) as T,
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
                
            case "authenticated":
                return {
                    values: baseFormData as T,
                    errors: {},
                    touched: {},
                    isDirty: false,
                    isSubmitting: false,
                    isValid: true,
                    authState: {
                        isAuthenticated: true,
                        isLoading: false,
                        user: {
                            id: this.generateId(),
                            handle: "johndoe",
                            name: "John Doe",
                            email: "john@example.com",
                        },
                    },
                };
                
            default:
                throw new Error(`Unknown form state scenario: ${scenario}`);
        }
    }
    
    /**
     * Create React Hook Form instance with pre-configured data
     */
    createFormInstance<T extends UserRegistrationFormData | UserProfileFormData | UserLoginFormData>(
        formType: "registration" | "profile" | "login",
        initialData?: Partial<T>,
    ): UseFormReturn<T> {
        let defaultValues: any;
        let resolver: any;
        
        switch (formType) {
            case "registration":
                defaultValues = {
                    ...this.createRegistrationFormData("empty"),
                    ...initialData,
                };
                resolver = yupResolver(this.createRegistrationSchema());
                break;
                
            case "profile":
                defaultValues = {
                    ...this.createProfileFormData("minimal"),
                    ...initialData,
                };
                resolver = yupResolver(this.createProfileSchema());
                break;
                
            case "login":
                defaultValues = {
                    ...this.createLoginFormData("empty"),
                    ...initialData,
                };
                resolver = yupResolver(this.createLoginSchema());
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
    async validateFormData<T extends UserRegistrationFormData | UserProfileFormData>(
        formData: T,
        formType: "registration" | "profile",
    ): Promise<{
        isValid: boolean;
        errors?: Record<string, string>;
        apiInput?: UserCreateInput | UserUpdateInput;
    }> {
        try {
            // Validate with form schema
            const schema = formType === "registration" 
                ? this.createRegistrationSchema()
                : this.createProfileSchema();
                
            await schema.validate(formData, { abortEarly: false });
            
            // Transform to API input and validate
            const apiInput = formType === "registration"
                ? this.transformRegistrationToAPIInput(formData as UserRegistrationFormData)
                : this.transformProfileToAPIInput(formData as UserProfileFormData);
                
            if (formType === "registration") {
                await userValidation.create.validate(apiInput);
            } else {
                await userValidation.update.validate(apiInput);
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
     * Transform registration form data to API input
     */
    private transformRegistrationToAPIInput(formData: UserRegistrationFormData): UserCreateInput {
        return {
            id: this.generateId(),
            email: formData.email,
            password: formData.password,
            handle: formData.handle,
            name: formData.name,
            bio: formData.bio,
            theme: formData.theme,
        };
    }
    
    /**
     * Transform profile form data to API input
     */
    private transformProfileToAPIInput(formData: UserProfileFormData): UserUpdateInput {
        return {
            id: this.generateId(),
            handle: formData.handle,
            name: formData.name,
            bio: formData.bio,
            theme: formData.theme,
            isPrivate: formData.isPrivate,
            isPrivateApis: formData.isPrivateApis,
            isPrivateProjects: formData.isPrivateProjects,
            isPrivateRoutines: formData.isPrivateRoutines,
            isPrivatePullRequests: formData.isPrivatePullRequests,
            isPrivateTeams: formData.isPrivateTeams,
        };
    }
    
    /**
     * Calculate password strength
     */
    calculatePasswordStrength(password: string): UserFormState["passwordStrength"] {
        let score = 0;
        const feedback: string[] = [];
        
        if (password.length >= 8) score++;
        else feedback.push("Use at least 8 characters");
        
        if (password.length >= 12) score++;
        
        if (/[a-z]/.test(password) && /[A-Z]/.test(password)) score++;
        else feedback.push("Include both uppercase and lowercase letters");
        
        if (/\d/.test(password)) score++;
        else feedback.push("Include at least one number");
        
        if (/[@$!%*?&]/.test(password)) score++;
        else feedback.push("Include at least one special character");
        
        return {
            score: Math.min(score, 4),
            feedback,
            isStrong: score >= 3,
        };
    }
}

/**
 * Form interaction simulator for user forms
 */
export class UserFormInteractionSimulator {
    private interactionDelay = 100;
    
    constructor(delay?: number) {
        this.interactionDelay = delay || 100;
    }
    
    /**
     * Simulate complete registration flow
     */
    async simulateRegistrationFlow(
        formInstance: UseFormReturn<UserRegistrationFormData>,
        formData: UserRegistrationFormData,
    ): Promise<void> {
        // Type email
        await this.simulateTyping(formInstance, "email", formData.email);
        await new Promise(resolve => setTimeout(resolve, 300));
        
        // Type password with strength feedback
        for (let i = 1; i <= formData.password.length; i++) {
            const partialPassword = formData.password.substring(0, i);
            await this.fillField(formInstance, "password", partialPassword);
            
            // Simulate password strength calculation
            const factory = new UserFormDataFactory();
            const strength = factory.calculatePasswordStrength(partialPassword);
            
            await new Promise(resolve => setTimeout(resolve, 50));
        }
        
        // Type confirm password
        await this.simulateTyping(formInstance, "confirmPassword", formData.confirmPassword);
        
        // Type handle with availability check simulation
        await this.simulateTyping(formInstance, "handle", formData.handle);
        await new Promise(resolve => setTimeout(resolve, 500)); // Simulate availability check
        
        // Type name
        await this.simulateTyping(formInstance, "name", formData.name);
        
        // Check agreements
        await this.fillField(formInstance, "agreeToTerms", formData.agreeToTerms);
        await new Promise(resolve => setTimeout(resolve, 200));
        
        await this.fillField(formInstance, "agreeToPrivacyPolicy", formData.agreeToPrivacyPolicy);
        await new Promise(resolve => setTimeout(resolve, 200));
        
        // Optional fields
        if (formData.bio) {
            await this.simulateTyping(formInstance, "bio", formData.bio);
        }
        
        if (formData.marketingEmails !== undefined) {
            await this.fillField(formInstance, "marketingEmails", formData.marketingEmails);
        }
    }
    
    /**
     * Simulate login flow
     */
    async simulateLoginFlow(
        formInstance: UseFormReturn<UserLoginFormData>,
        formData: UserLoginFormData,
    ): Promise<void> {
        // Type email or handle
        await this.simulateTyping(formInstance, "emailOrHandle", formData.emailOrHandle);
        await new Promise(resolve => setTimeout(resolve, 200));
        
        // Type password
        await this.simulateTyping(formInstance, "password", formData.password, { shouldMask: true });
        
        // Check remember me
        if (formData.rememberMe !== undefined) {
            await this.fillField(formInstance, "rememberMe", formData.rememberMe);
        }
    }
    
    /**
     * Simulate typing with optional masking
     */
    private async simulateTyping(
        formInstance: UseFormReturn<any>,
        fieldName: string,
        text: string,
        options?: { shouldMask?: boolean },
    ): Promise<void> {
        const { shouldMask = false } = options || {};
        
        for (let i = 1; i <= text.length; i++) {
            const partialText = shouldMask 
                ? "*".repeat(i - 1) + text.charAt(i - 1)
                : text.substring(0, i);
                
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
export const userFormFactory = new UserFormDataFactory();
export const userFormSimulator = new UserFormInteractionSimulator();

// Export pre-configured scenarios
export const userFormScenarios = {
    // Registration scenarios
    emptyRegistration: () => userFormFactory.createFormState("pristine", "registration"),
    validRegistration: () => userFormFactory.createFormState("valid", "registration"),
    registrationWithErrors: () => userFormFactory.createFormState("withErrors", "registration"),
    submittingRegistration: () => userFormFactory.createFormState("submitting", "registration"),
    
    // Profile scenarios
    minimalProfile: () => userFormFactory.createProfileFormData("minimal"),
    completeProfile: () => userFormFactory.createProfileFormData("complete"),
    privateProfile: () => userFormFactory.createProfileFormData("privateProfile"),
    
    // Login scenarios
    emptyLogin: () => userFormFactory.createLoginFormData("empty"),
    emailLogin: () => userFormFactory.createLoginFormData("withEmail"),
    handleLogin: () => userFormFactory.createLoginFormData("withHandle"),
    rememberMeLogin: () => userFormFactory.createLoginFormData("rememberMe"),
    
    // Interactive workflows
    async completeRegistrationWorkflow(formInstance: UseFormReturn<UserRegistrationFormData>) {
        const simulator = new UserFormInteractionSimulator();
        const formData = userFormFactory.createRegistrationFormData("complete");
        await simulator.simulateRegistrationFlow(formInstance, formData);
        return formData;
    },
    
    async quickLoginWorkflow(formInstance: UseFormReturn<UserLoginFormData>) {
        const simulator = new UserFormInteractionSimulator();
        const formData = userFormFactory.createLoginFormData("rememberMe");
        await simulator.simulateLoginFlow(formInstance, formData);
        return formData;
    },
};
