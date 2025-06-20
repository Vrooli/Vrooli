import { ResourceUsedFor } from "../../../shape/configs/base.js";
import { type NoteVersionConfigObject } from "../../../shape/configs/note.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures, mergeWithBaseDefaults } from "./baseConfigFixtures.js";

/**
 * Note configuration fixtures for testing note version settings
 */
export const noteConfigFixtures: ConfigTestFixtures<NoteVersionConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
    },

    complete: {
        __version: LATEST_CONFIG_VERSION,
        resources: [
            {
                link: "https://example.com/note-docs",
                usedFor: ResourceUsedFor.Notes,
                translations: [{
                    language: "en",
                    name: "Note Documentation",
                    description: "Complete guide for this note",
                }],
            },
            {
                link: "https://example.com/related",
                usedFor: ResourceUsedFor.Related,
                translations: [{
                    language: "en",
                    name: "Related Resources",
                    description: "Additional information related to this note",
                }],
            },
            {
                link: "https://github.com/example/note-project",
                usedFor: ResourceUsedFor.Developer,
                translations: [{
                    language: "en",
                    name: "Source Code",
                    description: "GitHub repository for this note project",
                }],
            },
        ],
    },

    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        resources: [],
    },

    invalid: {
        missingVersion: {
            // Missing required __version field
            resources: [],
        },
        invalidVersion: {
            __version: "0.1", // Invalid version
            resources: [],
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            resources: "not an array", // Should be array
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: 123, // Should be string
                    usedFor: "InvalidResourceType",
                    translations: [],
                },
            ],
        },
    },

    variants: {
        researchNote: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: "https://scholar.google.com/search?q=topic",
                    usedFor: ResourceUsedFor.Researching,
                    translations: [{
                        language: "en",
                        name: "Research Papers",
                        description: "Academic papers related to this topic",
                    }],
                },
                {
                    link: "https://arxiv.org/abs/2024.12345",
                    usedFor: ResourceUsedFor.Context,
                    translations: [{
                        language: "en",
                        name: "Background Paper",
                        description: "Foundational research paper",
                    }],
                },
            ],
        },

        tutorialNote: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: "https://youtube.com/watch?v=abc123",
                    usedFor: ResourceUsedFor.Tutorial,
                    translations: [{
                        language: "en",
                        name: "Video Tutorial",
                        description: "Step-by-step video guide",
                    }],
                },
                {
                    link: "https://example.com/tutorial",
                    usedFor: ResourceUsedFor.Learning,
                    translations: [{
                        language: "en",
                        name: "Written Tutorial",
                        description: "Detailed written instructions",
                    }],
                },
            ],
        },

        communityNote: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: "https://discord.gg/example",
                    usedFor: ResourceUsedFor.Community,
                    translations: [{
                        language: "en",
                        name: "Discord Server",
                        description: "Join our community discussion",
                    }],
                },
                {
                    link: "https://twitter.com/example",
                    usedFor: ResourceUsedFor.Social,
                    translations: [{
                        language: "en",
                        name: "Twitter Account",
                        description: "Follow for updates",
                    }],
                },
                {
                    link: "https://example.com/feed.rss",
                    usedFor: ResourceUsedFor.Feed,
                    translations: [{
                        language: "en",
                        name: "RSS Feed",
                        description: "Subscribe to updates",
                    }],
                },
            ],
        },

        multiLanguageNote: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: "https://example.com/guide",
                    usedFor: ResourceUsedFor.OfficialWebsite,
                    translations: [
                        {
                            language: "en",
                            name: "Official Guide",
                            description: "Complete documentation",
                        },
                        {
                            language: "es",
                            name: "Guía Oficial",
                            description: "Documentación completa",
                        },
                        {
                            language: "fr",
                            name: "Guide Officiel",
                            description: "Documentation complète",
                        },
                        {
                            language: "de",
                            name: "Offizieller Leitfaden",
                            description: "Vollständige Dokumentation",
                        },
                    ],
                },
            ],
        },

        projectNote: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: "https://github.com/org/project",
                    usedFor: ResourceUsedFor.Developer,
                    translations: [{
                        language: "en",
                        name: "Project Repository",
                        description: "Main development repository",
                    }],
                },
                {
                    link: "https://org.github.io/project",
                    usedFor: ResourceUsedFor.OfficialWebsite,
                    translations: [{
                        language: "en",
                        name: "Project Website",
                        description: "Official project documentation",
                    }],
                },
                {
                    link: "https://github.com/org/project/issues/new",
                    usedFor: ResourceUsedFor.Proposal,
                    translations: [{
                        language: "en",
                        name: "Submit Proposal",
                        description: "Propose new features or changes",
                    }],
                },
            ],
        },

        externalServiceNote: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: "https://api.example.com/v1",
                    usedFor: ResourceUsedFor.ExternalService,
                    translations: [{
                        language: "en",
                        name: "API Endpoint",
                        description: "External API service",
                    }],
                },
                {
                    link: "https://example.com/install",
                    usedFor: ResourceUsedFor.Install,
                    translations: [{
                        language: "en",
                        name: "Installation Guide",
                        description: "How to install and configure",
                    }],
                },
                {
                    link: "https://cal.example.com/schedule",
                    usedFor: ResourceUsedFor.Scheduling,
                    translations: [{
                        language: "en",
                        name: "Schedule Meeting",
                        description: "Book a consultation",
                    }],
                },
            ],
        },

        minimalResourceNote: {
            __version: LATEST_CONFIG_VERSION,
            resources: [{
                link: "https://example.com/note",
                usedFor: ResourceUsedFor.Notes,
                translations: [{
                    language: "en",
                    name: "Note Link",
                    // No description - testing minimal translation
                }],
            }],
        },
    },
};

/**
 * Create a note config with specific resources
 */
export function createNoteConfigWithResources(
    resources: Array<{
        link: string;
        usedFor: ResourceUsedFor;
        name: string;
        description?: string;
        language?: string;
    }>,
): NoteVersionConfigObject {
    return mergeWithBaseDefaults<NoteVersionConfigObject>({
        resources: resources.map(resource => ({
            link: resource.link,
            usedFor: resource.usedFor,
            translations: [{
                language: resource.language || "en",
                name: resource.name,
                description: resource.description,
            }],
        })),
    });
}

/**
 * Create a note config for a specific use case
 */
export function createNoteConfigForUseCase(
    useCase: "research" | "tutorial" | "community" | "project",
): NoteVersionConfigObject {
    const useCaseMap = {
        research: noteConfigFixtures.variants.researchNote,
        tutorial: noteConfigFixtures.variants.tutorialNote,
        community: noteConfigFixtures.variants.communityNote,
        project: noteConfigFixtures.variants.projectNote,
    };

    return useCaseMap[useCase];
}

/**
 * Create a note config with all resource types
 */
export function createNoteConfigWithAllResourceTypes(): NoteVersionConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        resources: Object.values(ResourceUsedFor).map((usedFor) => ({
            link: `https://example.com/${usedFor.toLowerCase()}`,
            usedFor,
            translations: [{
                language: "en",
                name: `${usedFor} Resource`,
                description: `Example ${usedFor} resource for testing`,
            }],
        })),
    };
}
