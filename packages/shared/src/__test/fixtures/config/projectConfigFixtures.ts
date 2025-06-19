import { type ProjectVersionConfigObject } from "../../../shape/configs/project.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { ResourceUsedFor } from "../../../shape/configs/base.js";
import { type ConfigTestFixtures, mergeWithBaseDefaults } from "./baseConfigFixtures.js";

/**
 * Project configuration fixtures for testing project version settings
 */
export const projectConfigFixtures: ConfigTestFixtures<ProjectVersionConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
    },
    
    complete: {
        __version: LATEST_CONFIG_VERSION,
        resources: [
            {
                link: "https://example.com/project-docs",
                usedFor: ResourceUsedFor.OfficialWebsite,
                translations: [{
                    language: "en",
                    name: "Project Documentation",
                    description: "Official documentation for this project"
                }]
            },
            {
                link: "https://github.com/example/project",
                usedFor: ResourceUsedFor.Developer,
                translations: [{
                    language: "en",
                    name: "Source Code",
                    description: "Project repository on GitHub"
                }]
            },
            {
                link: "https://example.com/tutorial",
                usedFor: ResourceUsedFor.Tutorial,
                translations: [{
                    language: "en",
                    name: "Getting Started",
                    description: "Step-by-step tutorial for beginners"
                }]
            }
        ]
    },
    
    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        resources: []
    },
    
    invalid: {
        missingVersion: {
            // Missing __version
            resources: [{
                link: "https://example.com",
                usedFor: ResourceUsedFor.Learning,
                translations: [{
                    language: "en",
                    name: "Resource"
                }]
            }]
        },
        invalidVersion: {
            __version: "0.1", // Invalid version
            resources: []
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            resources: "not an array", // Wrong type
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: 123, // Should be string
                    usedFor: "InvalidResourceType",
                    translations: "not an array" // Should be array
                }
            ]
        }
    },
    
    variants: {
        minimalWithResource: {
            __version: LATEST_CONFIG_VERSION,
            resources: [{
                link: "https://project.example.com",
                usedFor: ResourceUsedFor.OfficialWebsite,
                translations: [{
                    language: "en",
                    name: "Project Home"
                }]
            }]
        },
        
        openSourceProject: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: "https://github.com/opensource/project",
                    usedFor: ResourceUsedFor.Developer,
                    translations: [{
                        language: "en",
                        name: "GitHub Repository",
                        description: "Open source project repository"
                    }]
                },
                {
                    link: "https://opensource-project.readthedocs.io",
                    usedFor: ResourceUsedFor.OfficialWebsite,
                    translations: [{
                        language: "en",
                        name: "Documentation",
                        description: "Full project documentation"
                    }]
                },
                {
                    link: "https://discord.gg/opensource",
                    usedFor: ResourceUsedFor.Community,
                    translations: [{
                        language: "en",
                        name: "Discord Community",
                        description: "Join our community chat"
                    }]
                },
                {
                    link: "https://opencollective.com/project",
                    usedFor: ResourceUsedFor.Donation,
                    translations: [{
                        language: "en",
                        name: "Support Us",
                        description: "Help fund development"
                    }]
                }
            ]
        },
        
        educationalProject: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: "https://learn.example.com/course",
                    usedFor: ResourceUsedFor.Learning,
                    translations: [{
                        language: "en",
                        name: "Interactive Course",
                        description: "Learn by doing with our interactive course"
                    }]
                },
                {
                    link: "https://youtube.com/playlist?list=PLxxx",
                    usedFor: ResourceUsedFor.Tutorial,
                    translations: [{
                        language: "en",
                        name: "Video Tutorials",
                        description: "Complete video tutorial series"
                    }]
                },
                {
                    link: "https://example.com/exercises",
                    usedFor: ResourceUsedFor.Learning,
                    translations: [{
                        language: "en",
                        name: "Practice Exercises",
                        description: "Hands-on exercises and challenges"
                    }]
                }
            ]
        },
        
        researchProject: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: "https://arxiv.org/abs/2024.12345",
                    usedFor: ResourceUsedFor.Researching,
                    translations: [{
                        language: "en",
                        name: "Research Paper",
                        description: "Published research findings"
                    }]
                },
                {
                    link: "https://example.edu/project-data",
                    usedFor: ResourceUsedFor.Researching,
                    translations: [{
                        language: "en",
                        name: "Dataset",
                        description: "Research data and benchmarks"
                    }]
                },
                {
                    link: "https://example.com/project-notes",
                    usedFor: ResourceUsedFor.Notes,
                    translations: [{
                        language: "en",
                        name: "Research Notes",
                        description: "Detailed research methodology and notes"
                    }]
                }
            ]
        },
        
        commercialProject: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: "https://product.example.com",
                    usedFor: ResourceUsedFor.OfficialWebsite,
                    translations: [{
                        language: "en",
                        name: "Product Website",
                        description: "Official product information"
                    }]
                },
                {
                    link: "https://api.example.com/v1",
                    usedFor: ResourceUsedFor.ExternalService,
                    translations: [{
                        language: "en",
                        name: "API Service",
                        description: "Production API endpoint"
                    }]
                },
                {
                    link: "https://calendly.com/project-demo",
                    usedFor: ResourceUsedFor.Scheduling,
                    translations: [{
                        language: "en",
                        name: "Schedule Demo",
                        description: "Book a product demonstration"
                    }]
                }
            ]
        },
        
        multiLanguageProject: {
            __version: LATEST_CONFIG_VERSION,
            resources: [{
                link: "https://international.example.com",
                usedFor: ResourceUsedFor.OfficialWebsite,
                translations: [
                    {
                        language: "en",
                        name: "Project Website",
                        description: "Main project website"
                    },
                    {
                        language: "es",
                        name: "Sitio web del proyecto",
                        description: "Sitio web principal del proyecto"
                    },
                    {
                        language: "fr",
                        name: "Site Web du projet",
                        description: "Site Web principal du projet"
                    },
                    {
                        language: "de",
                        name: "Projekt-Website",
                        description: "Haupt-Projektwebsite"
                    },
                    {
                        language: "ja",
                        name: "プロジェクトウェブサイト",
                        description: "メインプロジェクトウェブサイト"
                    }
                ]
            }]
        },
        
        socialProject: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: "https://twitter.com/projecthandle",
                    usedFor: ResourceUsedFor.Social,
                    translations: [{
                        language: "en",
                        name: "Twitter/X",
                        description: "Follow us for updates"
                    }]
                },
                {
                    link: "https://linkedin.com/company/project",
                    usedFor: ResourceUsedFor.Social,
                    translations: [{
                        language: "en",
                        name: "LinkedIn",
                        description: "Professional updates and networking"
                    }]
                },
                {
                    link: "https://medium.com/@project",
                    usedFor: ResourceUsedFor.Feed,
                    translations: [{
                        language: "en",
                        name: "Blog",
                        description: "Project blog and announcements"
                    }]
                }
            ]
        }
    }
};

/**
 * Create a project config with specific resource types
 */
export function createProjectConfigWithResources(
    resources: Array<{
        link: string;
        usedFor: ResourceUsedFor;
        name: string;
        description?: string;
        language?: string;
    }>
): ProjectVersionConfigObject {
    return mergeWithBaseDefaults<ProjectVersionConfigObject>({
        resources: resources.map(resource => ({
            link: resource.link,
            usedFor: resource.usedFor,
            translations: [{
                language: resource.language || "en",
                name: resource.name,
                description: resource.description
            }]
        }))
    });
}

/**
 * Create a project config for a specific project type
 */
export function createProjectConfigByType(
    type: "opensource" | "educational" | "research" | "commercial",
    additionalResources: ProjectVersionConfigObject["resources"] = []
): ProjectVersionConfigObject {
    const baseConfigs = {
        opensource: projectConfigFixtures.variants.openSourceProject,
        educational: projectConfigFixtures.variants.educationalProject,
        research: projectConfigFixtures.variants.researchProject,
        commercial: projectConfigFixtures.variants.commercialProject
    };
    
    const baseConfig = baseConfigs[type];
    return {
        ...baseConfig,
        resources: [...(baseConfig.resources || []), ...additionalResources]
    };
}