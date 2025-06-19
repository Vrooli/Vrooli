// Note: Project types don't have specific CreateInput/UpdateInput types
// Projects use form-specific data structures before being transformed

/**
 * Form data fixtures for project-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Project creation form data
 */
export const minimalProjectCreateFormInput = {
    name: "New Project",
    description: "A basic project description",
    isPrivate: false,
    handle: "new-project",
};

export const completeProjectCreateFormInput = {
    name: "AI Assistant Platform",
    description: "A comprehensive platform for building AI-powered assistants with advanced natural language processing capabilities",
    handle: "ai-assistant-platform",
    isPrivate: false,
    tags: ["ai", "nlp", "assistant", "platform"],
    labels: ["mvp", "in-progress"],
    versionLabel: "1.0.0",
    versionNotes: "Initial release with core features",
    goals: [
        "Build conversational AI interface",
        "Implement plugin system",
        "Create developer SDK",
    ],
    resourceLinks: [
        {
            title: "Documentation",
            url: "https://docs.example.com",
            usedFor: "Documentation",
        },
        {
            title: "API Reference",
            url: "https://api.example.com",
            usedFor: "Api",
        },
    ],
    team: "team_123456789", // Team ID if creating for team
};

/**
 * Project update form data
 */
export const projectUpdateFormInput = {
    name: "AI Assistant Platform v2",
    description: "Enhanced platform with multi-modal capabilities and improved performance. Now supports voice, text, and image inputs.",
    versionLabel: "2.0.0",
    versionNotes: "Major update with voice support and performance improvements",
    tags: ["ai", "nlp", "voice", "multimodal"],
    labels: ["stable", "production"],
    isComplete: true,
    completedAt: "2024-01-20T10:00:00Z",
};

/**
 * Project version form data
 */
export const createVersionFormInput = {
    versionLabel: "1.1.0",
    versionNotes: "Bug fixes and performance improvements",
    changes: [
        "Fixed memory leak in chat handler",
        "Improved response time by 30%",
        "Added support for custom plugins",
    ],
    isPrivate: false,
    isComplete: false,
};

export const publishVersionFormInput = {
    versionId: "version_123456789",
    isComplete: true,
    releaseNotes: "This version includes major improvements:\n- New UI design\n- Better performance\n- Bug fixes",
    notifyFollowers: true,
};

/**
 * Project directory form data
 */
export const projectDirectoryFormInput = {
    directories: [
        {
            name: "src",
            description: "Source code",
            children: [
                { name: "components", description: "React components" },
                { name: "services", description: "Business logic" },
                { name: "utils", description: "Utility functions" },
            ],
        },
        {
            name: "docs",
            description: "Documentation",
            children: [
                { name: "api", description: "API documentation" },
                { name: "guides", description: "User guides" },
            ],
        },
        {
            name: "tests",
            description: "Test files",
            children: [],
        },
    ],
};

/**
 * Project resource management form data
 */
export const addResourceFormInput = {
    title: "Tutorial Video",
    url: "https://youtube.com/watch?v=example",
    usedFor: "Tutorial",
    description: "Step-by-step guide for getting started",
    listId: "list_123456789", // Resource list ID
};

export const resourceListFormInput = {
    name: "Learning Resources",
    description: "Curated list of learning materials",
    resources: [
        {
            title: "Getting Started Guide",
            url: "https://docs.example.com/start",
            usedFor: "Documentation",
        },
        {
            title: "Video Tutorial",
            url: "https://youtube.com/watch?v=123",
            usedFor: "Tutorial",
        },
        {
            title: "Example Repository",
            url: "https://github.com/example/repo",
            usedFor: "ExampleRepository",
        },
    ],
};

/**
 * Project collaboration form data
 */
export const projectTransferFormInput = {
    newOwner: "user_123456789", // User or Team ID
    ownerType: "User", // "User" | "Team"
    reason: "Handing over project maintenance",
    keepAccess: true, // Keep contributor access after transfer
    confirmText: "TRANSFER",
};

export const projectPermissionsFormInput = {
    public: {
        canView: true,
        canFork: true,
        canComment: true,
        canReport: true,
    },
    members: {
        canEdit: true,
        canDelete: false,
        canTransfer: false,
        canManagePermissions: false,
    },
};

/**
 * Project import/export form data
 */
export const importProjectFormInput = {
    source: "github", // "github" | "gitlab" | "upload"
    repositoryUrl: "https://github.com/username/repo",
    branch: "main",
    includeHistory: false,
    visibility: "public", // "public" | "private" | "unlisted"
};

export const exportProjectFormInput = {
    format: "zip", // "zip" | "github" | "json"
    includeVersions: true,
    includeResources: true,
    includeMembers: false,
    includeComments: false,
};

/**
 * Form validation states
 */
export const projectFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            name: "", // Required but empty
            handle: "invalid handle!", // Invalid format
            versionLabel: "v1", // Invalid version format
        },
        errors: {
            name: "Project name is required",
            handle: "Handle can only contain letters, numbers, and hyphens",
            versionLabel: "Version must follow semantic versioning (e.g., 1.0.0)",
        },
        touched: {
            name: true,
            handle: true,
            versionLabel: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalProjectCreateFormInput,
        errors: {},
        touched: {
            name: true,
            description: true,
            handle: true,
            isPrivate: true,
        },
        isValid: true,
        isSubmitting: false,
    },
};

/**
 * Helper function to create project form initial values
 */
export const createProjectFormInitialValues = (projectData?: Partial<any>) => ({
    name: projectData?.name || "",
    description: projectData?.description || "",
    handle: projectData?.handle || "",
    isPrivate: projectData?.isPrivate || false,
    tags: projectData?.tags || [],
    labels: projectData?.labels || [],
    versionLabel: projectData?.versionLabel || "1.0.0",
    versionNotes: projectData?.versionNotes || "",
    ...projectData,
});

/**
 * Helper function to validate semantic version
 */
export const validateSemanticVersion = (version: string): string | null => {
    const semverRegex = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$/;
    
    if (!version) return "Version is required";
    if (!semverRegex.test(version)) {
        return "Version must follow semantic versioning (e.g., 1.0.0)";
    }
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformProjectFormToApiInput = (formData: any) => ({
    ...formData,
    // Convert tag strings to proper format
    tags: formData.tags?.map((tag: string) => ({ tag })) || [],
    // Convert label strings to proper format
    labels: formData.labels?.map((label: string) => ({ label })) || [],
    // Filter out empty resource links
    resourceLinks: formData.resourceLinks?.filter((link: any) => link.url) || [],
    // Convert goals array to structured format
    goals: formData.goals?.map((goal: string, index: number) => ({
        text: goal,
        order: index,
    })) || [],
});

/**
 * Mock suggestions for project forms
 */
export const mockProjectSuggestions = {
    tags: [
        { tag: "javascript", count: 5432 },
        { tag: "typescript", count: 4321 },
        { tag: "react", count: 3210 },
        { tag: "nodejs", count: 2109 },
    ],
    labels: [
        { label: "bug", color: "#ff0000" },
        { label: "feature", color: "#00ff00" },
        { label: "documentation", color: "#0000ff" },
        { label: "help-wanted", color: "#ffaa00" },
    ],
    versionSuggestions: [
        "1.0.0",
        "0.1.0",
        "2.0.0-beta.1",
        "1.1.0-rc.1",
    ],
};