import type { BotCreateInput, BotUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for bot-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Bot creation form data
 */
export const minimalBotCreateFormInput: Partial<BotCreateInput> = {
    name: "Test Bot",
    handle: "testbot",
    isBotDepictingPerson: false,
    isPrivate: false,
};

export const completeBotCreateFormInput = {
    name: "AI Assistant Bot",
    handle: "ai-assistant",
    isBotDepictingPerson: false,
    isPrivate: false,
    bio: "An advanced AI assistant designed to help with various tasks including coding, research, and creative writing",
    profileImage: null, // File object would go here
    bannerImage: null, // File object would go here
    botSettings: {
        model: "gpt-4",
        temperature: 0.7,
        maxTokens: 2048,
        systemPrompt: "You are a helpful AI assistant. Be concise, accurate, and friendly in your responses.",
        assistantId: "",
        customConfig: {
            responseStyle: "professional",
            specialties: ["coding", "research", "writing"],
            languages: ["en", "es", "fr"],
        },
    },
};

/**
 * Bot depicting person form data
 */
export const botDepictingPersonFormInput = {
    name: "Virtual Customer Support Agent",
    handle: "support-agent",
    isBotDepictingPerson: true,
    isPrivate: false,
    bio: "I'm a virtual representation of our customer support team, here to help you 24/7",
    profileImage: null, // File object would go here
    botSettings: {
        model: "gpt-3.5-turbo",
        temperature: 0.5,
        maxTokens: 1024,
        systemPrompt: "You are a friendly customer support representative. Always be helpful and empathetic.",
        personality: "friendly",
        background: "10+ years of customer service experience",
        department: "Customer Support",
    },
};

/**
 * Bot update form data
 */
export const minimalBotUpdateFormInput = {
    name: "Updated Bot Name",
    bio: "Updated bot description",
};

export const completeBotUpdateFormInput = {
    name: "Enhanced AI Assistant",
    handle: "enhanced-ai",
    isBotDepictingPerson: false,
    isPrivate: true,
    bio: "An enhanced version of our AI assistant with improved capabilities in data analysis, code generation, and creative tasks",
    profileImage: null, // File object for new image
    bannerImage: null, // File object for new image
    botSettings: {
        model: "claude-3-opus",
        temperature: 0.8,
        maxTokens: 4096,
        systemPrompt: "You are an advanced AI assistant with expertise in multiple domains. Provide detailed, well-structured responses.",
        assistantId: "asst_987654321",
        customConfig: {
            responseStyle: "detailed",
            specialties: ["coding", "data-analysis", "creative-writing", "research"],
            languages: ["en", "es", "fr", "de", "ja"],
            features: {
                codeExecution: true,
                webSearch: true,
                imageGeneration: false,
            },
        },
    },
};

/**
 * Bot settings form data variants
 */
export const botSettingsVariants = {
    minimal: {
        model: "gpt-3.5-turbo",
        temperature: 0.7,
    },
    creative: {
        model: "gpt-4",
        temperature: 0.9,
        maxTokens: 2048,
        systemPrompt: "You are a creative writing assistant. Be imaginative and help users with storytelling, poetry, and creative projects.",
    },
    technical: {
        model: "claude-3-opus",
        temperature: 0.3,
        maxTokens: 4096,
        systemPrompt: "You are a technical expert. Provide precise, accurate information about programming, engineering, and technical topics.",
        customConfig: {
            codeHighlighting: true,
            syntaxValidation: true,
            languages: ["python", "javascript", "rust", "go"],
        },
    },
    educational: {
        model: "gpt-4",
        temperature: 0.5,
        maxTokens: 2048,
        systemPrompt: "You are an educational tutor. Explain concepts clearly, provide examples, and adapt to the student's learning level.",
        customConfig: {
            gradeLevel: "college",
            subjects: ["math", "science", "history", "literature"],
            teachingStyle: "socratic",
        },
    },
};

/**
 * Private/Public bot variants
 */
export const privateBotFormInput = {
    name: "Private Research Bot",
    handle: "research-bot",
    isBotDepictingPerson: false,
    isPrivate: true,
    bio: "A private bot for internal research and development",
    botSettings: {
        model: "gpt-4",
        temperature: 0.7,
        maxTokens: 4096,
        accessLevel: "restricted",
        allowedUsers: [],
    },
};

export const publicBotFormInput = {
    name: "Community Helper Bot",
    handle: "community-bot",
    isBotDepictingPerson: false,
    isPrivate: false,
    bio: "A public bot available to help the entire community with general questions",
    botSettings: {
        model: "gpt-3.5-turbo",
        temperature: 0.7,
        maxTokens: 1024,
        rateLimit: 100,
        cooldownPeriod: 60,
    },
};

/**
 * Form validation states
 */
export const botFormValidationStates = {
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
            handle: "ab", // Too short (min 3 chars)
            isBotDepictingPerson: false,
            botSettings: {
                model: "", // Required but empty
            },
        },
        errors: {
            name: "Bot name is required",
            handle: "Handle must be at least 3 characters",
            "botSettings.model": "Model selection is required",
        },
        touched: {
            name: true,
            handle: true,
            "botSettings.model": true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalBotCreateFormInput,
        errors: {},
        touched: {
            name: true,
            handle: true,
            isBotDepictingPerson: true,
            isPrivate: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeBotCreateFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create bot form initial values
 */
export const createBotFormInitialValues = (botData?: Partial<any>) => ({
    name: botData?.name || "",
    handle: botData?.handle || "",
    bio: botData?.bio || "",
    isBotDepictingPerson: botData?.isBotDepictingPerson || false,
    isPrivate: botData?.isPrivate || false,
    botSettings: botData?.botSettings || {
        model: "",
        temperature: 0.7,
        maxTokens: 2048,
    },
    ...botData,
});

/**
 * Helper function to validate bot handle
 */
export const validateBotHandle = (handle: string): string | null => {
    if (!handle) return "Handle is required";
    if (handle.length < 3) return "Handle must be at least 3 characters";
    if (handle.length > 16) return "Handle must be less than 16 characters";
    if (!/^[a-zA-Z0-9-]+$/.test(handle)) {
        return "Handle can only contain letters, numbers, and hyphens";
    }
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformBotFormToApiInput = (formData: any) => ({
    ...formData,
    // Convert file objects to upload format
    profileImage: formData.profileImage?.file || undefined,
    bannerImage: formData.bannerImage?.file || undefined,
    // Ensure botSettings is an object
    botSettings: formData.botSettings || {},
    // Add translations if bio is provided
    translations: formData.bio ? {
        create: [{
            id: "trans_" + Date.now(),
            language: "en",
            bio: formData.bio,
        }],
    } : undefined,
});

/**
 * Mock model options for form selects
 */
export const mockModelOptions = [
    { value: "gpt-3.5-turbo", label: "GPT-3.5 Turbo", description: "Fast and efficient for most tasks" },
    { value: "gpt-4", label: "GPT-4", description: "Most capable model for complex tasks" },
    { value: "claude-3-opus", label: "Claude 3 Opus", description: "Advanced reasoning and analysis" },
    { value: "claude-3-sonnet", label: "Claude 3 Sonnet", description: "Balanced performance and speed" },
    { value: "llama-2-70b", label: "Llama 2 70B", description: "Open source alternative" },
];

/**
 * Mock temperature presets
 */
export const mockTemperaturePresets = {
    precise: { value: 0.3, label: "Precise", description: "More focused and deterministic" },
    balanced: { value: 0.7, label: "Balanced", description: "Good for most use cases" },
    creative: { value: 0.9, label: "Creative", description: "More varied and creative outputs" },
};

/**
 * Mock file inputs for testing
 */
export const mockBotFileInputs = {
    profileImage: {
        file: null, // Would be File object
        preview: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==",
        name: "bot-avatar.png",
        size: 12345,
        type: "image/png",
    },
    bannerImage: {
        file: null, // Would be File object
        preview: "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAAAAA...",
        name: "bot-banner.jpg",
        size: 67890,
        type: "image/jpeg",
    },
};

/**
 * Mock autocomplete suggestions
 */
export const mockBotSuggestions = {
    handles: [
        "helper-bot",
        "assistant-ai",
        "support-bot",
        "research-bot",
    ],
    systemPrompts: [
        "You are a helpful assistant.",
        "You are an expert in your field.",
        "You are a friendly and knowledgeable guide.",
        "You are a creative problem solver.",
    ],
    specialties: [
        "coding",
        "writing",
        "research",
        "data-analysis",
        "translation",
        "tutoring",
        "customer-support",
    ],
};