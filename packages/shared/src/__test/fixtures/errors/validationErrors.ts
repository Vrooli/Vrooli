/**
 * Validation error fixtures for testing field-level validation errors
 * 
 * These fixtures provide common validation error scenarios including
 * form errors, nested validation, and complex field validation patterns.
 */

export interface ValidationError {
    fields: Record<string, string | string[]>;
    message?: string;
}

export interface NestedValidationError {
    [key: string]: string | string[] | NestedValidationError | NestedValidationError[];
}

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
        } satisfies ValidationError,

        login: {
            fields: {
                email: "Invalid email or password",
                password: "Invalid email or password",
            },
        } satisfies ValidationError,

        profile: {
            fields: {
                bio: "Bio cannot exceed 500 characters",
                website: "Please enter a valid URL",
                phone: "Please enter a valid phone number",
            },
        } satisfies ValidationError,

        project: {
            fields: {
                name: "Project name is required",
                description: "Description must be at least 10 characters",
                tags: "Maximum 10 tags allowed",
                isPrivate: "Privacy setting is required",
            },
        } satisfies ValidationError,
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
        } satisfies ValidationError,

        emails: {
            fields: {
                emails: ["Invalid email format", undefined, "Email domain not allowed"] as any,
            },
            message: "One or more emails are invalid",
        } satisfies ValidationError,

        schedule: {
            fields: {
                exceptions: ["Date must be in the future", "Duplicate exception date"],
                recurrences: ["Invalid recurrence pattern", "End date before start date"],
            },
        } satisfies ValidationError,
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
        } satisfies ValidationError,

        crossFieldValidation: {
            fields: {
                startDate: "Start date must be before end date",
                endDate: "End date must be after start date",
                duration: "Duration doesn't match date range",
            },
        } satisfies ValidationError,

        businessRules: {
            fields: {
                credits: "Insufficient credits for this operation",
                teamSize: "Team size exceeds plan limit",
                apiCalls: "API call limit exceeded for current plan",
            },
            message: "Please upgrade your plan to continue",
        } satisfies ValidationError,
    },

    // Factory functions
    factories: {
        /**
         * Create a simple field validation error
         */
        createFieldError: (fieldName: string, message: string): ValidationError => ({
            fields: {
                [fieldName]: message,
            },
        }),

        /**
         * Create multiple field errors
         */
        createMultiFieldError: (errors: Record<string, string>): ValidationError => ({
            fields: errors,
        }),

        /**
         * Create array field errors
         */
        createArrayFieldError: (fieldName: string, errors: (string | undefined)[]): ValidationError => ({
            fields: {
                [fieldName]: errors as any,
            },
        }),

        /**
         * Create nested validation errors
         */
        createNestedError: (structure: NestedValidationError): NestedValidationError => structure,

        /**
         * Create a validation error with custom message
         */
        createValidationError: (fields: Record<string, string | string[]>, message?: string): ValidationError => ({
            fields,
            ...(message && { message }),
        }),
    },
};