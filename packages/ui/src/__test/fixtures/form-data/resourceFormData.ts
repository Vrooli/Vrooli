import { ResourceUsedFor, type ResourceShape } from "@vrooli/shared";

/**
 * Form data fixtures for resource creation and editing
 * These represent data as it appears in form state before submission
 * 
 * Resources are links or shortcuts that provide additional context or functionality.
 * They can be added to resource lists and are categorized by their usage purpose.
 */

/**
 * Minimal resource form input - just a basic link
 */
export const minimalResourceFormInput = {
    link: "https://example.com",
    usedFor: ResourceUsedFor.Context,
    resourceType: "link" as const,
    translations: [
        {
            language: "en",
            name: "Example Resource",
            description: "",
        },
    ],
};

/**
 * Complete resource form input with full details
 */
export const completeResourceFormInput = {
    link: "https://docs.example.com/api/v2",
    usedFor: ResourceUsedFor.Developer,
    resourceType: "link" as const,
    translations: [
        {
            language: "en",
            name: "API Documentation v2",
            description: "Comprehensive API documentation with examples and best practices for developers",
        },
        {
            language: "es",
            name: "Documentación de API v2",
            description: "Documentación completa de API con ejemplos y mejores prácticas para desarrolladores",
        },
    ],
};

/**
 * Shortcut resource form input (internal link)
 */
export const shortcutResourceFormInput = {
    link: "/dashboard/projects",
    usedFor: ResourceUsedFor.Context,
    resourceType: "shortcut" as const,
    selectedShortcut: {
        label: "Projects",
        value: "/dashboard/projects",
        icon: "Project",
    },
    translations: [
        {
            language: "en",
            name: "Projects Dashboard",
            description: "Quick access to your projects dashboard",
        },
    ],
};

/**
 * Resource form variants for different use cases
 */
export const resourceFormVariants = {
    community: {
        link: "https://discord.gg/community",
        usedFor: ResourceUsedFor.Community,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Community Discord",
                description: "Join our Discord server to connect with other users",
            },
        ],
    },
    tutorial: {
        link: "https://youtube.com/watch?v=tutorial123",
        usedFor: ResourceUsedFor.Tutorial,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Getting Started Tutorial",
                description: "Step-by-step video guide for beginners",
            },
        ],
    },
    documentation: {
        link: "https://docs.project.com",
        usedFor: ResourceUsedFor.OfficialWebsite,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Official Documentation",
                description: "Complete user guide and API reference",
            },
        ],
    },
    social: {
        link: "https://twitter.com/project",
        usedFor: ResourceUsedFor.Social,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Twitter Profile",
                description: "Follow us for the latest updates and announcements",
            },
        ],
    },
    donation: {
        link: "https://opencollective.com/project",
        usedFor: ResourceUsedFor.Donation,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Support the Project",
                description: "Help support the development of this project",
            },
        ],
    },
    externalService: {
        link: "https://api.external-service.com",
        usedFor: ResourceUsedFor.ExternalService,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "External API",
                description: "Third-party service integration endpoint",
            },
        ],
    },
    learning: {
        link: "https://learn.example.com/course/advanced",
        usedFor: ResourceUsedFor.Learning,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Advanced Course",
                description: "Comprehensive learning materials for advanced users",
            },
        ],
    },
    proposal: {
        link: "https://github.com/project/rfcs/123",
        usedFor: ResourceUsedFor.Proposal,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "RFC: New Feature Proposal",
                description: "Request for comments on proposed new feature",
            },
        ],
    },
    feed: {
        link: "https://blog.project.com/rss",
        usedFor: ResourceUsedFor.Feed,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Project Blog RSS",
                description: "Stay updated with our latest blog posts",
            },
        ],
    },
    install: {
        link: "https://releases.project.com/latest",
        usedFor: ResourceUsedFor.Install,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Download Latest Release",
                description: "Get the latest stable version of the software",
            },
        ],
    },
    notes: {
        link: "https://notes.project.com/meeting-2024-01",
        usedFor: ResourceUsedFor.Notes,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Meeting Notes January 2024",
                description: "Notes from the January 2024 planning meeting",
            },
        ],
    },
    related: {
        link: "https://similar-project.com",
        usedFor: ResourceUsedFor.Related,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Similar Project",
                description: "Related project that might be of interest",
            },
        ],
    },
    researching: {
        link: "https://research-paper.com/ai-automation",
        usedFor: ResourceUsedFor.Researching,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "AI Automation Research",
                description: "Research paper on AI automation techniques",
            },
        ],
    },
    scheduling: {
        link: "https://calendar.project.com/events",
        usedFor: ResourceUsedFor.Scheduling,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Event Calendar",
                description: "Project events and important dates",
            },
        ],
    },
};

/**
 * Resource form with multiple languages
 */
export const multiLanguageResourceFormInput = {
    link: "https://github.com/project/repo",
    usedFor: ResourceUsedFor.Developer,
    resourceType: "link" as const,
    translations: [
        {
            language: "en",
            name: "GitHub Repository",
            description: "Main project repository with source code and documentation",
        },
        {
            language: "es",
            name: "Repositorio de GitHub",
            description: "Repositorio principal del proyecto con código fuente y documentación",
        },
        {
            language: "fr",
            name: "Dépôt GitHub",
            description: "Dépôt principal du projet avec code source et documentation",
        },
        {
            language: "de",
            name: "GitHub-Repository",
            description: "Haupt-Projekt-Repository mit Quellcode und Dokumentation",
        },
    ],
};

/**
 * Form validation test cases
 */
export const invalidResourceFormInputs = {
    missingLink: {
        link: "", // Invalid: empty link
        usedFor: ResourceUsedFor.Context,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Valid Name",
                description: "Valid description",
            },
        ],
    },
    invalidUrl: {
        link: "not-a-valid-url", // Invalid: malformed URL
        usedFor: ResourceUsedFor.Context,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Valid Name",
                description: "Valid description",
            },
        ],
    },
    missingTranslations: {
        link: "https://example.com",
        usedFor: ResourceUsedFor.Context,
        resourceType: "link" as const,
        translations: [], // Invalid: no translations
    },
    emptyTranslationName: {
        link: "https://example.com",
        usedFor: ResourceUsedFor.Context,
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "", // Invalid: empty name
                description: "Valid description",
            },
        ],
    },
    invalidLanguage: {
        link: "https://example.com",
        usedFor: ResourceUsedFor.Context,
        resourceType: "link" as const,
        translations: [
            {
                // @ts-expect-error - Testing invalid input
                language: "", // Invalid: empty language
                name: "Valid Name",
                description: "Valid description",
            },
        ],
    },
    missingUsedFor: {
        link: "https://example.com",
        // @ts-expect-error - Testing invalid input
        usedFor: undefined, // Invalid: missing usedFor
        resourceType: "link" as const,
        translations: [
            {
                language: "en",
                name: "Valid Name",
                description: "Valid description",
            },
        ],
    },
    shortcutWithoutSelection: {
        link: "",
        usedFor: ResourceUsedFor.Context,
        resourceType: "shortcut" as const,
        selectedShortcut: null, // Invalid: shortcut type but no selection
        translations: [
            {
                language: "en",
                name: "Shortcut Name",
                description: "Shortcut description",
            },
        ],
    },
};

/**
 * Form validation states for testing
 */
export const resourceFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            link: "invalid-url",
            usedFor: ResourceUsedFor.Context,
            translations: [
                {
                    language: "en",
                    name: "", // Empty name triggers error
                    description: "",
                },
            ],
        },
        errors: {
            link: "Please enter a valid URL",
            "translations.0.name": "Resource name is required",
        },
        touched: {
            link: true,
            "translations.0.name": true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalResourceFormInput,
        errors: {},
        touched: {
            link: true,
            usedFor: true,
            "translations.0.name": true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeResourceFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create resource form initial values
 */
export const createResourceFormInitialValues = (resourceData?: Partial<any>) => ({
    link: resourceData?.link || "",
    usedFor: resourceData?.usedFor || ResourceUsedFor.Context,
    resourceType: resourceData?.resourceType || "link",
    selectedShortcut: resourceData?.selectedShortcut || null,
    translations: resourceData?.translations || [
        {
            language: "en",
            name: "",
            description: "",
        },
    ],
    ...resourceData,
});

/**
 * Helper function to transform form data to ResourceShape format
 */
export const transformResourceFormToShape = (formData: any): ResourceShape => ({
    __typename: "Resource",
    id: "DUMMY_ID",
    index: 0,
    link: formData.link,
    usedFor: formData.usedFor,
    list: {
        __typename: "ResourceList",
        id: "DUMMY_ID",
        listFor: { __typename: "RoutineVersion", id: "DUMMY_ID" },
    },
    translations: formData.translations.map((t: any, index: number) => ({
        __typename: "ResourceTranslation",
        id: `DUMMY_ID_${index}`,
        language: t.language,
        name: t.name,
        description: t.description || "",
    })),
});

/**
 * Mock shortcuts for testing shortcut resource type
 */
export const mockShortcuts = [
    {
        label: "Dashboard",
        value: "/dashboard",
        icon: "Dashboard",
    },
    {
        label: "Projects",
        value: "/dashboard/projects",
        icon: "Project",
    },
    {
        label: "Routines",
        value: "/dashboard/routines",
        icon: "Routine",
    },
    {
        label: "Notes",
        value: "/dashboard/notes",
        icon: "Note",
    },
    {
        label: "Settings",
        value: "/settings",
        icon: "Settings",
    },
    {
        label: "Profile",
        value: "/profile",
        icon: "User",
    },
];

/**
 * Common URL patterns for testing
 */
export const commonUrlPatterns = {
    github: "https://github.com/username/repository",
    youtube: "https://youtube.com/watch?v=example123",
    documentation: "https://docs.example.com/getting-started",
    discord: "https://discord.gg/community",
    twitter: "https://twitter.com/username",
    linkedin: "https://linkedin.com/company/example",
    website: "https://example.com",
    api: "https://api.example.com/v1",
    blog: "https://blog.example.com",
    forum: "https://forum.example.com",
    wiki: "https://wiki.example.com",
    support: "https://support.example.com",
};