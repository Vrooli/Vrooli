import type { ResourceVersionCreateInput, ResourceVersionUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for resource version-related forms
 * These represent data as it appears in form state before submission
 * 
 * Resource versions manage different iterations of resources (APIs, routines, standards, etc.)
 * with versioning capabilities, configuration options, and translation support.
 */

/**
 * Minimal resource version form input - basic version with required fields
 */
export const minimalResourceVersionFormInput: Partial<ResourceVersionCreateInput> = {
    versionLabel: "1.0.0",
    isComplete: false,
    isPrivate: false,
    translations: [
        {
            language: "en",
            name: "Basic Resource Version",
            description: "",
        },
    ],
};

/**
 * Complete resource version form input with all features
 */
export const completeResourceVersionFormInput = {
    versionLabel: "2.1.0",
    versionNotes: "Major update with new features and bug fixes. Includes improved configuration options and automation capabilities.",
    codeLanguage: "typescript",
    config: {
        theme: "dark",
        autoSave: true,
        version: "2.0",
        features: ["automation", "validation"],
        settings: { timeout: 30000, retries: 3 },
    },
    isAutomatable: true,
    isComplete: true,
    isInternal: false,
    isPrivate: false,
    resourceSubType: "RoutineApi",
    translations: [
        {
            language: "en",
            name: "Complete Resource Version",
            description: "A comprehensive resource version with all features enabled and advanced configuration options.",
            details: "This version includes automation capabilities, custom configuration, and multi-language support for enhanced functionality.",
            instructions: "Follow the setup guide carefully and ensure all dependencies are installed before using automation features.",
        },
        {
            language: "es",
            name: "Versión Completa del Recurso",
            description: "Una versión integral del recurso con todas las características habilitadas y opciones de configuración avanzadas.",
            details: "Esta versión incluye capacidades de automatización, configuración personalizada y soporte multiidioma.",
        },
    ],
};

/**
 * Resource version form variants for different types
 */
export const resourceVersionFormVariants = {
    routineApi: {
        versionLabel: "1.0.0",
        resourceSubType: "RoutineApi",
        codeLanguage: "javascript",
        isAutomatable: true,
        isComplete: true,
        translations: [
            {
                language: "en",
                name: "API Routine Version",
                description: "Automated routine with API integration capabilities",
            },
        ],
    },
    routineGenerate: {
        versionLabel: "1.0.0",
        resourceSubType: "RoutineGenerate",
        isAutomatable: true,
        isComplete: true,
        config: { generationType: "content", outputFormat: "markdown" },
        translations: [
            {
                language: "en",
                name: "Content Generation Routine",
                description: "Automated routine for generating content",
            },
        ],
    },
    codeConverter: {
        versionLabel: "1.0.0",
        resourceSubType: "CodeDataConverter",
        codeLanguage: "python",
        isAutomatable: false,
        isComplete: true,
        config: { inputFormat: "json", outputFormat: "csv" },
        translations: [
            {
                language: "en",
                name: "Data Converter Tool",
                description: "Python script for converting data between formats",
            },
        ],
    },
    standardPrompt: {
        versionLabel: "1.0.0",
        resourceSubType: "StandardPrompt",
        isAutomatable: false,
        isComplete: true,
        translations: [
            {
                language: "en",
                name: "Standard Prompt Template",
                description: "Reusable prompt template for consistent AI interactions",
            },
        ],
    },
    informational: {
        versionLabel: "1.0.0",
        resourceSubType: "Informational",
        isAutomatable: false,
        isComplete: true,
        translations: [
            {
                language: "en",
                name: "Documentation Resource",
                description: "Informational resource with guidelines and documentation",
            },
        ],
    },
};

/**
 * Resource version with different privacy settings
 */
export const privateResourceVersionFormInput = {
    versionLabel: "0.9.0-beta",
    versionNotes: "Private beta version for internal testing",
    isPrivate: true,
    isInternal: true,
    isComplete: false,
    translations: [
        {
            language: "en",
            name: "Private Beta Version",
            description: "Internal testing version not yet ready for public release",
            details: "This is a pre-release version for internal evaluation and testing purposes only.",
        },
    ],
};

export const publicResourceVersionFormInput = {
    versionLabel: "2.0.0",
    versionNotes: "Public release with stable features and comprehensive documentation",
    isPrivate: false,
    isInternal: false,
    isComplete: true,
    translations: [
        {
            language: "en",
            name: "Public Release Version",
            description: "Stable public release with full feature set and documentation",
            details: "Production-ready version with complete feature set and extensive testing.",
            instructions: "This version is ready for production use. See documentation for setup instructions.",
        },
    ],
};

/**
 * Resource version update form data
 */
export const updateResourceVersionFormInput = {
    versionLabel: "2.1.1",
    versionNotes: "Patch release with minor fixes and improvements",
    config: {
        theme: "light",
        autoSave: false,
        version: "2.1",
        bugFixes: true,
    },
    isComplete: true,
    translations: [
        {
            language: "en",
            name: "Updated Resource Version",
            description: "Updated version with bug fixes and minor improvements",
        },
    ],
};

/**
 * Multi-language resource version form data
 */
export const multiLanguageResourceVersionFormInput = {
    versionLabel: "1.5.0",
    versionNotes: "Internationalization update with support for multiple languages",
    isComplete: true,
    translations: [
        {
            language: "en",
            name: "Multi-language Resource Version",
            description: "Resource version with comprehensive multi-language support",
            details: "Complete internationalization support with translations for major languages.",
            instructions: "Select your preferred language from the settings menu to use localized content.",
        },
        {
            language: "es",
            name: "Versión Multiidioma del Recurso",
            description: "Versión del recurso con soporte integral para múltiples idiomas",
            details: "Soporte completo de internacionalización con traducciones para idiomas principales.",
            instructions: "Selecciona tu idioma preferido desde el menú de configuración para usar contenido localizado.",
        },
        {
            language: "fr",
            name: "Version Multilingue de la Ressource",
            description: "Version de ressource avec support complet multilingue",
            details: "Support d'internationalisation complet avec traductions pour les langues principales.",
            instructions: "Sélectionnez votre langue préférée dans le menu des paramètres pour utiliser le contenu localisé.",
        },
        {
            language: "de",
            name: "Mehrsprachige Ressourcenversion",
            description: "Ressourcenversion mit umfassender mehrsprachiger Unterstützung",
            details: "Vollständige Internationalisierungsunterstützung mit Übersetzungen für Hauptsprachen.",
        },
    ],
};

/**
 * Form validation test cases
 */
export const invalidResourceVersionFormInputs = {
    missingVersionLabel: {
        versionLabel: "", // Invalid: empty version label
        isComplete: false,
        translations: [
            {
                language: "en",
                name: "Valid Name",
                description: "Valid description",
            },
        ],
    },
    invalidSemanticVersion: {
        versionLabel: "v1.0", // Invalid: not semantic version format
        isComplete: false,
        translations: [
            {
                language: "en",
                name: "Valid Name",
                description: "Valid description",
            },
        ],
    },
    missingTranslations: {
        versionLabel: "1.0.0",
        isComplete: false,
        translations: [], // Invalid: no translations
    },
    emptyTranslationName: {
        versionLabel: "1.0.0",
        isComplete: false,
        translations: [
            {
                language: "en",
                name: "", // Invalid: empty name
                description: "Valid description",
            },
        ],
    },
    invalidLanguage: {
        versionLabel: "1.0.0",
        isComplete: false,
        translations: [
            {
                // @ts-expect-error - Testing invalid input
                language: "", // Invalid: empty language
                name: "Valid Name",
                description: "Valid description",
            },
        ],
    },
    invalidCodeLanguage: {
        versionLabel: "1.0.0",
        codeLanguage: "x".repeat(130), // Invalid: too long
        isComplete: false,
        translations: [
            {
                language: "en",
                name: "Valid Name",
                description: "Valid description",
            },
        ],
    },
    invalidConfig: {
        versionLabel: "1.0.0",
        // @ts-expect-error - Testing invalid input
        config: "not-an-object", // Invalid: should be object
        isComplete: false,
        translations: [
            {
                language: "en",
                name: "Valid Name",
                description: "Valid description",
            },
        ],
    },
    longVersionNotes: {
        versionLabel: "1.0.0",
        versionNotes: "x".repeat(4100), // Invalid: too long
        isComplete: false,
        translations: [
            {
                language: "en",
                name: "Valid Name",
                description: "Valid description",
            },
        ],
    },
    invalidResourceSubType: {
        versionLabel: "1.0.0",
        // @ts-expect-error - Testing invalid input
        resourceSubType: "InvalidSubType", // Invalid: not a valid enum value
        isComplete: false,
        translations: [
            {
                language: "en",
                name: "Valid Name",
                description: "Valid description",
            },
        ],
    },
};

/**
 * Form validation states for testing
 */
export const resourceVersionFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            versionLabel: "v1", // Invalid format
            codeLanguage: "x".repeat(130), // Too long
            translations: [
                {
                    language: "en",
                    name: "", // Empty name triggers error
                    description: "",
                },
            ],
        },
        errors: {
            versionLabel: "Version must follow semantic versioning (e.g., 1.0.0)",
            codeLanguage: "Code language must be 128 characters or less",
            "translations.0.name": "Resource name is required",
        },
        touched: {
            versionLabel: true,
            codeLanguage: true,
            "translations.0.name": true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalResourceVersionFormInput,
        errors: {},
        touched: {
            versionLabel: true,
            isComplete: true,
            "translations.0.name": true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeResourceVersionFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create resource version form initial values
 */
export const createResourceVersionFormInitialValues = (versionData?: Partial<any>) => ({
    versionLabel: versionData?.versionLabel || "1.0.0",
    versionNotes: versionData?.versionNotes || "",
    codeLanguage: versionData?.codeLanguage || "",
    config: versionData?.config || {},
    isAutomatable: versionData?.isAutomatable || false,
    isComplete: versionData?.isComplete || false,
    isInternal: versionData?.isInternal || false,
    isPrivate: versionData?.isPrivate || false,
    resourceSubType: versionData?.resourceSubType || "",
    translations: versionData?.translations || [
        {
            language: "en",
            name: "",
            description: "",
            details: "",
            instructions: "",
        },
    ],
    ...versionData,
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
export const transformResourceVersionFormToApiInput = (formData: any) => ({
    ...formData,
    // Ensure config is an object
    config: formData.config && typeof formData.config === "object" ? formData.config : {},
    // Transform translations to create format
    translationsCreate: formData.translations?.map((t: any, index: number) => ({
        id: `trans_${Date.now()}_${index}`,
        language: t.language,
        name: t.name,
        description: t.description || "",
        details: t.details || "",
        instructions: t.instructions || "",
    })) || [],
    // Remove form-specific fields
    translations: undefined,
});

/**
 * Mock resource sub-type options
 */
export const mockResourceSubTypeOptions = [
    { value: "RoutineApi", label: "API Routine", description: "Automated routine with API integration" },
    { value: "RoutineGenerate", label: "Generation Routine", description: "Content generation routine" },
    { value: "RoutineInformational", label: "Informational Routine", description: "Documentation and guidance routine" },
    { value: "CodeDataConverter", label: "Data Converter", description: "Code for data transformation" },
    { value: "CodeSmartContract", label: "Smart Contract", description: "Blockchain smart contract code" },
    { value: "StandardPrompt", label: "Prompt Template", description: "Reusable AI prompt template" },
    { value: "StandardInformational", label: "Information Standard", description: "Documentation standard or guideline" },
    { value: "Informational", label: "General Information", description: "General informational resource" },
];

/**
 * Mock code language options
 */
export const mockCodeLanguageOptions = [
    { value: "javascript", label: "JavaScript" },
    { value: "typescript", label: "TypeScript" },
    { value: "python", label: "Python" },
    { value: "java", label: "Java" },
    { value: "go", label: "Go" },
    { value: "rust", label: "Rust" },
    { value: "solidity", label: "Solidity" },
    { value: "sql", label: "SQL" },
    { value: "bash", label: "Bash" },
    { value: "json", label: "JSON" },
    { value: "yaml", label: "YAML" },
    { value: "markdown", label: "Markdown" },
];

/**
 * Mock configuration templates
 */
export const mockConfigurationTemplates = {
    basic: {
        version: "1.0",
        features: [],
    },
    automation: {
        version: "1.0",
        features: ["automation", "scheduling"],
        settings: {
            timeout: 30000,
            retries: 3,
            autoSave: true,
        },
    },
    development: {
        version: "dev",
        features: ["debugging", "testing"],
        environment: "development",
        debugging: true,
    },
    production: {
        version: "1.0",
        features: ["optimization", "monitoring"],
        environment: "production",
        optimization: true,
        monitoring: {
            enabled: true,
            level: "info",
        },
    },
};

/**
 * Common version label suggestions
 */
export const mockVersionSuggestions = [
    "1.0.0",
    "0.1.0",
    "2.0.0-beta.1",
    "1.1.0-rc.1",
    "0.9.0-alpha",
    "1.0.1",
    "2.1.0",
    "3.0.0-preview.1",
];