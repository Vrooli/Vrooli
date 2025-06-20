/**
 * Validation error fixtures for testing field-level validation errors
 * 
 * These fixtures provide common validation error scenarios including
 * form errors, nested validation, and complex field validation patterns.
 */

import {
    BaseErrorFactory,
    type EnhancedValidationError,
    type ErrorContext
} from "./types.js";

export interface ValidationError {
    fields: Record<string, string | string[]>;
    message?: string;
}

export interface NestedValidationError {
    [key: string]: string | string[] | NestedValidationError | NestedValidationError[];
}

/**
 * ValidationErrorFactory - Factory for creating enhanced validation errors
 * 
 * Validation errors are always blocking since they prevent form submission
 * and require user correction.
 */
export class ValidationErrorFactory extends BaseErrorFactory<EnhancedValidationError> {
    standard: EnhancedValidationError = {
        fields: {
            email: "Invalid email format",
            password: "Password is required",
        },
        message: "Please correct the errors below",
        userImpact: "blocking",
        recovery: {
            strategy: "fail",
        },
    };

    withDetails: EnhancedValidationError = {
        fields: {
            email: "Please enter a valid email address",
            password: "Password must be at least 8 characters and contain uppercase, lowercase, numbers, and symbols",
            confirmPassword: "Passwords do not match",
        },
        message: "Form validation failed",
        userImpact: "blocking",
        recovery: {
            strategy: "fail",
        },
        fieldPath: "user.credentials",
        constraint: "password.strength",
        suggestion: "Use a password manager to generate a strong password",
    };

    variants = {
        minimal: {
            fields: { field: "Invalid value" },
            userImpact: "blocking" as const,
            recovery: { strategy: "fail" as const },
        } satisfies EnhancedValidationError,

        complex: {
            fields: {
                startDate: "Start date must be before end date",
                endDate: "End date must be after start date",
                duration: "Duration doesn't match date range",
            },
            message: "Date validation failed",
            userImpact: "blocking" as const,
            recovery: { strategy: "fail" as const },
            fieldPath: "schedule.dates",
            constraint: "date.ordering",
            suggestion: "Select an end date that is after the start date",
        } satisfies EnhancedValidationError,

        nested: {
            fields: {
                "user.email": "Email is required",
                "user.profile.bio": "Bio cannot exceed 500 characters",
                "settings.notifications.email": "Invalid notification preferences",
            },
            message: "Multiple validation errors occurred",
            userImpact: "blocking" as const,
            recovery: { strategy: "fail" as const },
        } satisfies EnhancedValidationError,
    };

    create(overrides?: Partial<EnhancedValidationError>): EnhancedValidationError {
        return {
            ...this.standard,
            ...overrides,
        };
    }

    createWithContext(context: ErrorContext): EnhancedValidationError {
        return {
            ...this.standard,
            context,
            telemetry: {
                traceId: `trace-${Date.now()}`,
                spanId: `span-${Date.now()}`,
                tags: {
                    operation: context.operation || "validation",
                    resource: context.resource?.type || "form",
                },
                level: "warn",
            },
        };
    }

    /**
     * Create a field-specific validation error with helpful context
     */
    createFieldError(
        fieldName: string,
        message: string,
        options?: {
            constraint?: string;
            suggestion?: string;
            invalidValue?: any;
        },
    ): EnhancedValidationError {
        return {
            fields: { [fieldName]: message },
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: fieldName,
            ...options,
        };
    }

    /**
     * Create validation error for array fields
     */
    createArrayFieldError(
        fieldName: string,
        errors: (string | undefined)[],
    ): EnhancedValidationError {
        return {
            fields: { [fieldName]: errors as any },
            message: `${fieldName} validation failed`,
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: fieldName,
        };
    }
}

// Create factory instance
export const validationErrorFactory = new ValidationErrorFactory();

// Export enhanced validation scenarios for testing
export const enhancedValidationScenarios = {
    /**
     * Field validation with helpful suggestions
     */
    withSuggestions: {
        passwordStrength: validationErrorFactory.createFieldError(
            "password",
            "Password is too weak",
            {
                constraint: "password.strength",
                suggestion: "Include uppercase, lowercase, numbers, and special characters",
                invalidValue: "password123", // redacted in real use
            },
        ),

        emailFormat: validationErrorFactory.createFieldError(
            "email",
            "Invalid email format",
            {
                constraint: "email.format",
                suggestion: "Use format: name@domain.com",
                invalidValue: "user@",
            },
        ),

        urlProtocol: validationErrorFactory.createFieldError(
            "website",
            "URL must include protocol",
            {
                constraint: "url.protocol",
                suggestion: "Add https:// or http:// before the domain",
                invalidValue: "example.com",
            },
        ),
    },

    /**
     * Validation with telemetry context
     */
    withTelemetry: validationErrorFactory.createWithContext({
        user: { id: "user123", role: "standard" },
        operation: "profile.update",
        resource: { type: "user", id: "user123" },
        environment: "production",
    }),

    /**
     * Complex nested validation
     */
    deepNested: {
        fields: {
            "team.members[0].permissions.read": "Permission denied",
            "team.members[0].permissions.write": "Requires admin role",
            "team.settings.billing.card.expired": "Payment method expired",
            "team.settings.limits.storage": "Storage limit exceeded",
        },
        message: "Multiple configuration errors",
        userImpact: "blocking",
        recovery: { strategy: "fail" },
        fieldPath: "team.configuration",
        constraint: "nested.validation",
        suggestion: "Review team member permissions and billing information",
    } satisfies EnhancedValidationError,
};

export const validationErrorFixtures = {
    // Form validation errors
    formErrors: {
        registration: {
            fields: {
                email: "Email is required",
                password: "Password must be at least 8 characters",
                confirmPassword: "Passwords do not match",
                name: "Name must be between 2 and 50 characters",
            },
            message: "Please complete all required fields",
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: "registration",
            constraint: "required",
            suggestion: "Fill in all required fields to create your account",
        } satisfies EnhancedValidationError,

        login: {
            fields: {
                email: "Invalid email or password",
                password: "Invalid email or password",
            },
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: "credentials",
            constraint: "authentication",
            suggestion: "Check your email and password and try again",
        } satisfies EnhancedValidationError,

        profile: {
            fields: {
                bio: "Bio cannot exceed 500 characters",
                website: "Please enter a valid URL",
                phone: "Please enter a valid phone number",
            },
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: "profile",
            constraint: "format",
            suggestion: "Ensure all fields meet the format requirements",
        } satisfies EnhancedValidationError,

        project: {
            fields: {
                name: "Project name is required",
                description: "Description must be at least 10 characters",
                tags: "Maximum 10 tags allowed",
                isPrivate: "Privacy setting is required",
            },
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: "project",
            constraint: "limits",
            suggestion: "Provide a descriptive name and keep tags under the limit",
        } satisfies EnhancedValidationError,
    },

    // Field-specific errors
    fieldErrors: {
        email: {
            required: "Email is required",
            invalid: "Please enter a valid email address",
            taken: "This email is already registered",
            domain: "Email domain is not allowed",
        },

        password: {
            required: "Password is required",
            tooShort: "Password must be at least 8 characters",
            tooWeak: "Password is too weak. Include uppercase, lowercase, numbers, and symbols",
            noMatch: "Passwords do not match",
            common: "This password is too common",
        },

        username: {
            required: "Username is required",
            tooShort: "Username must be at least 3 characters",
            tooLong: "Username cannot exceed 30 characters",
            invalid: "Username can only contain letters, numbers, and underscores",
            taken: "This username is already taken",
        },

        url: {
            required: "URL is required",
            invalid: "Please enter a valid URL",
            protocol: "URL must start with http:// or https://",
            domain: "Invalid domain name",
        },
    },

    // Nested validation errors
    nested: {
        project: {
            name: "Project name is required",
            team: {
                id: "Invalid team ID",
                permissions: "Insufficient team permissions",
            },
            versions: [
                { versionNumber: "Version number must be unique" },
                { changelog: "Changelog is required for published versions" },
            ],
            tags: ["Tag name too long", "Duplicate tag"],
        } as NestedValidationError,

        routine: {
            name: "Routine name is required",
            description: "Description too short",
            steps: [
                {
                    name: "Step name is required",
                    type: "Invalid step type",
                    config: {
                        url: "Invalid URL format",
                        method: "HTTP method not allowed",
                    },
                },
                {
                    name: "Duplicate step name",
                    connections: ["Invalid connection reference"],
                },
            ],
        } as NestedValidationError,

        team: {
            name: "Team name is required",
            members: [
                {
                    userId: "User not found",
                    role: "Invalid role",
                },
                {
                    userId: "User already a member",
                    permissions: "Invalid permission set",
                },
            ],
            settings: {
                visibility: "Invalid visibility setting",
                joinPolicy: "Join policy not allowed for private teams",
            },
        } as NestedValidationError,
    },

    // Array validation errors
    arrayErrors: {
        tags: {
            fields: {
                tags: ["Tag name too short", "Duplicate tag: 'test'", "Invalid characters in tag"],
            },
            message: "Tag validation failed",
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: "tags",
            constraint: "array.items",
            suggestion: "Use unique tags with valid characters (letters, numbers, hyphens)",
        } satisfies EnhancedValidationError,

        emails: {
            fields: {
                emails: ["Invalid email format", undefined, "Email domain not allowed"] as any,
            },
            message: "One or more emails are invalid",
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: "emails",
            constraint: "array.format",
            suggestion: "Ensure all email addresses are valid and from allowed domains",
        } satisfies EnhancedValidationError,

        schedule: {
            fields: {
                exceptions: ["Date must be in the future", "Duplicate exception date"],
                recurrences: ["Invalid recurrence pattern", "End date before start date"],
            },
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: "schedule",
            constraint: "temporal",
            suggestion: "Check date ordering and ensure no duplicate exceptions",
        } satisfies EnhancedValidationError,
    },

    // Complex validation scenarios
    complex: {
        conditionalFields: {
            fields: {
                isPrivate: "Privacy setting is required",
                password: "Password is required for private resources",
                inviteOnly: "This option is only available for private resources",
            },
            message: "Please review the privacy settings",
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: "privacy",
            constraint: "conditional",
            suggestion: "Private resources require a password for access control",
        } satisfies EnhancedValidationError,

        crossFieldValidation: {
            fields: {
                startDate: "Start date must be before end date",
                endDate: "End date must be after start date",
                duration: "Duration doesn't match date range",
            },
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: "dates",
            constraint: "cross-field",
            suggestion: "Ensure dates are in chronological order and duration matches",
        } satisfies EnhancedValidationError,

        businessRules: {
            fields: {
                credits: "Insufficient credits for this operation",
                teamSize: "Team size exceeds plan limit",
                apiCalls: "API call limit exceeded for current plan",
            },
            message: "Please upgrade your plan to continue",
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: "limits",
            constraint: "business",
            suggestion: "Upgrade to a higher plan or reduce resource usage",
        } satisfies EnhancedValidationError,
    },

    // Factory functions (enhanced versions)
    factories: {
        /**
         * Create a simple field validation error
         */
        createFieldError: (fieldName: string, message: string): EnhancedValidationError => ({
            fields: {
                [fieldName]: message,
            },
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: fieldName,
        }),

        /**
         * Create multiple field errors
         */
        createMultiFieldError: (errors: Record<string, string>): EnhancedValidationError => ({
            fields: errors,
            userImpact: "blocking",
            recovery: { strategy: "fail" },
        }),

        /**
         * Create array field errors
         */
        createArrayFieldError: (fieldName: string, errors: (string | undefined)[]): EnhancedValidationError => ({
            fields: {
                [fieldName]: errors as any,
            },
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            fieldPath: fieldName,
        }),

        /**
         * Create nested validation errors
         */
        createNestedError: (structure: NestedValidationError): NestedValidationError => structure,

        /**
         * Create a validation error with custom message
         */
        createValidationError: (fields: Record<string, string | string[]>, message?: string): EnhancedValidationError => ({
            fields,
            ...(message && { message }),
            userImpact: "blocking",
            recovery: { strategy: "fail" },
        }),

        /**
         * Create a validation error with full enhancement
         */
        createEnhancedValidationError: (
            fields: Record<string, string | string[]>,
            options?: {
                message?: string;
                fieldPath?: string;
                invalidValue?: any;
                constraint?: string;
                suggestion?: string;
                context?: ErrorContext;
            },
        ): EnhancedValidationError => ({
            fields,
            userImpact: "blocking",
            recovery: { strategy: "fail" },
            ...options,
        }),
    },
};
