import type { ApiKeyExternalUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for external API key-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * External API key creation form data
 */
export const minimalApiKeyExternalCreateFormInput = {
    name: "OpenAI Development Key",
    service: "OpenAI",
    key: "sk-test-1234567890abcdef",
};

export const completeApiKeyExternalCreateFormInput = {
    name: "Production Anthropic API Key",
    service: "Anthropic",
    key: "sk-ant-api03-production-key-with-full-access",
    disabled: false,
    description: "Main production key for Claude AI integration",
    environment: "production",
    lastUsedAt: null,
    metadata: {
        project: "AI Assistant",
        team: "Backend",
        purpose: "Production LLM calls",
    },
};

/**
 * External API key update form data
 */
export const minimalApiKeyExternalUpdateFormInput = {
    name: "Updated External API Key",
};

export const completeApiKeyExternalUpdateFormInput = {
    name: "Production Claude API Key - Updated",
    service: "Anthropic",
    key: "sk-ant-api03-updated-production-key",
    disabled: false,
    description: "Updated with new access levels and rate limits",
    environment: "production",
    metadata: {
        project: "AI Platform v2",
        team: "ML Infrastructure",
        purpose: "Enhanced LLM integration",
        lastRotated: "2024-01-15",
    },
};

/**
 * External API key service-specific form data
 */
export const openAIKeyFormInput = {
    name: "OpenAI GPT-4 Key",
    service: "OpenAI",
    key: "sk-test-openai-gpt4-access",
    model: "gpt-4",
    organization: "org-123456789",
};

export const anthropicKeyFormInput = {
    name: "Claude 3 API Key",
    service: "Anthropic",
    key: "sk-ant-api03-claude3-access",
    model: "claude-3-opus",
    workspace: "prod-workspace",
};

export const googleKeyFormInput = {
    name: "Google Gemini Key",
    service: "Google",
    key: "AIzaSy-test-google-gemini-key",
    project: "my-gcp-project",
    region: "us-central1",
};

export const mistralKeyFormInput = {
    name: "Mistral API Key",
    service: "Mistral",
    key: "test-mistral-api-key-123456",
    tier: "premium",
};

/**
 * External API key rotation form data
 */
export const rotateApiKeyExternalFormInput = {
    currentKeyId: "123456789012345678",
    newKey: "sk-test-rotated-key-new-1234567890",
    confirmRotation: true,
    reason: "Regular security rotation",
    notifyServices: true,
    gracePeriodMinutes: 30,
};

/**
 * External API key disable/enable form data
 */
export const disableApiKeyExternalFormInput = {
    disabled: true,
    reason: "Suspicious activity detected",
    temporaryDuration: "24h",
};

export const enableApiKeyExternalFormInput = {
    disabled: false,
    verificationCode: "123456",
    acknowledgeRisks: true,
};

/**
 * External API key validation form data
 */
export const validateApiKeyExternalFormInput = {
    keyId: "123456789012345678",
    testEndpoint: true,
    testModel: "gpt-3.5-turbo",
    testPrompt: "Say hello",
};

/**
 * External API key deletion form data
 */
export const deleteApiKeyExternalFormInput = {
    confirmText: "DELETE",
    reason: "No longer needed",
    removeFromAllServices: true,
};

/**
 * Form validation states
 */
export const apiKeyExternalFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            name: "ab", // Too short
            service: "", // Required but empty
            key: "invalid", // Too short for most services
        },
        errors: {
            name: "Name must be at least 3 characters",
            service: "Service is required",
            key: "API key format is invalid",
        },
        touched: {
            name: true,
            service: true,
            key: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalApiKeyExternalCreateFormInput,
        errors: {},
        touched: {
            name: true,
            service: true,
            key: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: minimalApiKeyExternalCreateFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create external API key form initial values
 */
export const createApiKeyExternalFormInitialValues = (apiKeyData?: Partial<any>) => ({
    name: apiKeyData?.name || "",
    service: apiKeyData?.service || "",
    key: apiKeyData?.key || "",
    disabled: apiKeyData?.disabled || false,
    description: apiKeyData?.description || "",
    environment: apiKeyData?.environment || "development",
    metadata: apiKeyData?.metadata || {},
    ...apiKeyData,
});

/**
 * Helper function to transform form data to API format
 */
export const transformApiKeyExternalFormToApiInput = (formData: any) => ({
    ...formData,
    // Ensure service is properly formatted
    service: formData.service?.trim() || undefined,
    // Mask key if needed (for display purposes)
    displayKey: formData.key ? maskApiKey(formData.key) : undefined,
    // Convert metadata object to JSON string if needed
    metadata: formData.metadata ? JSON.stringify(formData.metadata) : undefined,
    // Remove UI-only fields
    confirmText: undefined,
    confirmRotation: undefined,
    acknowledgeRisks: undefined,
    verificationCode: undefined,
});

/**
 * Helper function to validate API key format by service
 */
export const validateApiKeyFormat = (key: string, service: string): string | null => {
    if (!key) return "API key is required";
    
    const validationRules: Record<string, { pattern: RegExp; message: string }> = {
        OpenAI: {
            pattern: /^sk-[a-zA-Z0-9]{48}$/,
            message: "OpenAI keys must start with 'sk-' followed by 48 characters",
        },
        Anthropic: {
            pattern: /^sk-ant-[a-zA-Z0-9-]{40,}$/,
            message: "Anthropic keys must start with 'sk-ant-' followed by API key",
        },
        Google: {
            pattern: /^AIza[a-zA-Z0-9-_]{35}$/,
            message: "Google API keys must start with 'AIza' followed by 35 characters",
        },
        Mistral: {
            pattern: /^[a-zA-Z0-9]{32,}$/,
            message: "Mistral API keys must be at least 32 characters",
        },
    };
    
    const rule = validationRules[service];
    if (!rule) {
        // Generic validation for unknown services
        if (key.length < 10) return "API key must be at least 10 characters";
        return null;
    }
    
    if (!rule.pattern.test(key)) {
        return rule.message;
    }
    
    return null;
};

/**
 * Helper function to mask API key for display
 */
export const maskApiKey = (key: string): string => {
    if (!key || key.length < 8) return key;
    
    const visibleStart = 6;
    const visibleEnd = 4;
    const maskedLength = key.length - visibleStart - visibleEnd;
    
    return `${key.slice(0, visibleStart)}${"*".repeat(Math.max(maskedLength, 4))}${key.slice(-visibleEnd)}`;
};

/**
 * Mock external API key display data
 */
export const mockApiKeyExternalDisplay = {
    masked: {
        key: "sk-tes****************************abcd",
        service: "OpenAI",
        lastUsed: "2024-01-15T10:30:00Z",
        createdAt: "2023-12-01T08:00:00Z",
        status: "active",
        rateLimitRemaining: 4500,
        rateLimitTotal: 5000,
    },
    revealed: {
        key: "sk-test-complete-1234567890abcdef1234567890abcdef",
        warning: "This key will only be shown once. Please copy it to a secure location.",
        copySuccess: false,
    },
};

/**
 * Mock suggestions for external API key forms
 */
export const mockApiKeyExternalSuggestions = {
    services: [
        { name: "OpenAI", icon: "openai", popular: true },
        { name: "Anthropic", icon: "anthropic", popular: true },
        { name: "Google", icon: "google", popular: true },
        { name: "Mistral", icon: "mistral", popular: false },
        { name: "Cohere", icon: "cohere", popular: false },
        { name: "HuggingFace", icon: "huggingface", popular: false },
    ],
    environments: [
        { value: "development", label: "Development" },
        { value: "staging", label: "Staging" },
        { value: "production", label: "Production" },
    ],
    models: {
        OpenAI: ["gpt-4", "gpt-3.5-turbo", "dall-e-3"],
        Anthropic: ["claude-3-opus", "claude-3-sonnet", "claude-3-haiku"],
        Google: ["gemini-pro", "gemini-pro-vision"],
        Mistral: ["mistral-large", "mistral-medium", "mistral-small"],
    },
};

/**
 * Test scenarios for external API keys
 */
export const apiKeyExternalTestScenarios = {
    multipleServices: [
        { ...openAIKeyFormInput, id: "key_openai_123" },
        { ...anthropicKeyFormInput, id: "key_anthropic_456" },
        { ...googleKeyFormInput, id: "key_google_789" },
    ],
    rotationInProgress: {
        oldKey: { ...openAIKeyFormInput, status: "rotating_out" },
        newKey: { ...openAIKeyFormInput, key: "sk-test-new-rotated-key", status: "rotating_in" },
        rotationEndsAt: "2024-01-20T10:00:00Z",
    },
    quotaExceeded: {
        ...openAIKeyFormInput,
        quotaStatus: {
            used: 5000,
            limit: 5000,
            resetsAt: "2024-02-01T00:00:00Z",
            isExceeded: true,
        },
    },
};