import { describe, it, expect, vi, beforeEach } from "vitest";
import { ProjectVersionConfig, type ProjectVersionConfigObject } from "./project.js";
import { type ResourceVersion } from "../../api/types.js";
import { ResourceUsedFor } from "./base.js";

describe("ProjectVersionConfig", () => {
    let mockLogger: any;

    beforeEach(() => {
        mockLogger = {
            trace: vi.fn(),
            debug: vi.fn(),
            info: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        };
    });

    describe("constructor", () => {
        it("should create ProjectVersionConfig with complete data", () => {
            const config: ProjectVersionConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://github.com/example/project",
                        usedFor: ResourceUsedFor.Developer,
                        translations: [
                            {
                                language: "en",
                                name: "Project Repository",
                                description: "Main project source code repository",
                            },
                        ],
                    },
                    {
                        link: "https://project-docs.com",
                        usedFor: ResourceUsedFor.OfficialWebsite,
                        translations: [
                            {
                                language: "en",
                                name: "Official Documentation",
                            },
                        ],
                    },
                ],
            };

            const projectConfig = new ProjectVersionConfig({ config });

            expect(projectConfig.__version).toBe("1.0");
            expect(projectConfig.resources).toHaveLength(2);
            expect(projectConfig.resources[0].link).toBe("https://github.com/example/project");
            expect(projectConfig.resources[0].usedFor).toBe(ResourceUsedFor.Developer);
            expect(projectConfig.resources[1].usedFor).toBe(ResourceUsedFor.OfficialWebsite);
        });

        it("should create ProjectVersionConfig with minimal data", () => {
            const config: ProjectVersionConfigObject = {
                __version: "1.0",
            };

            const projectConfig = new ProjectVersionConfig({ config });

            expect(projectConfig.__version).toBe("1.0");
            expect(projectConfig.resources).toEqual([]);
        });

        it("should create ProjectVersionConfig without version (uses default)", () => {
            const config: Partial<ProjectVersionConfigObject> = {
                resources: [],
            };

            const projectConfig = new ProjectVersionConfig({ config: config as ProjectVersionConfigObject });

            expect(projectConfig.__version).toBe("1.0");
            expect(projectConfig.resources).toEqual([]);
        });
    });

    describe("parse", () => {
        it("should parse valid project version data", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: {
                    __version: "1.0",
                    resources: [
                        {
                            link: "https://project.example.com",
                            translations: [
                                {
                                    language: "en",
                                    name: "Project Site",
                                },
                            ],
                        },
                    ],
                },
            };

            const projectConfig = ProjectVersionConfig.parse(versionData, mockLogger);

            expect(projectConfig.__version).toBe("1.0");
            expect(projectConfig.resources).toHaveLength(1);
            expect(projectConfig.resources[0].link).toBe("https://project.example.com");
        });

        it("should parse with null config", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: null as any,
            };

            const projectConfig = ProjectVersionConfig.parse(versionData, mockLogger);

            expect(projectConfig.__version).toBe("1.0");
            expect(projectConfig.resources).toEqual([]);
        });

        it("should parse with undefined config", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: undefined as any,
            };

            const projectConfig = ProjectVersionConfig.parse(versionData, mockLogger);

            expect(projectConfig.__version).toBe("1.0");
            expect(projectConfig.resources).toEqual([]);
        });

        it("should parse with useFallbacks parameter (no effect for ProjectVersionConfig)", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: {
                    __version: "1.0",
                },
            };

            const configWithFallbacks = ProjectVersionConfig.parse(versionData, mockLogger, { useFallbacks: true });
            const configWithoutFallbacks = ProjectVersionConfig.parse(versionData, mockLogger, { useFallbacks: false });

            // Both should be the same as ProjectVersionConfig doesn't have additional fallback properties
            expect(configWithFallbacks.export()).toEqual(configWithoutFallbacks.export());
        });
    });

    describe("default", () => {
        it("should create default ProjectVersionConfig", () => {
            const projectConfig = ProjectVersionConfig.default();

            expect(projectConfig.__version).toBe("1.0");
            expect(projectConfig.resources).toEqual([]);
        });
    });

    describe("export", () => {
        it("should export all properties", () => {
            const originalConfig: ProjectVersionConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://community.project.com",
                        usedFor: ResourceUsedFor.Community,
                        translations: [
                            {
                                language: "en",
                                name: "Community Forum",
                                description: "Discuss the project with other users",
                            },
                            {
                                language: "ja",
                                name: "コミュニティフォーラム",
                                description: "他のユーザーとプロジェクトについて話し合う",
                            },
                        ],
                    },
                ],
            };

            const projectConfig = new ProjectVersionConfig({ config: originalConfig });
            const exported = projectConfig.export();

            expect(exported).toEqual(originalConfig);
        });

        it("should export minimal config", () => {
            const projectConfig = new ProjectVersionConfig({ config: { __version: "1.0" } });
            const exported = projectConfig.export();

            expect(exported).toEqual({
                __version: "1.0",
                resources: [],
            });
        });
    });

    describe("Resource management (inherited from BaseConfig)", () => {
        it("should add resources", () => {
            const projectConfig = ProjectVersionConfig.default();

            projectConfig.addResource({
                link: "https://donate.project.com",
                usedFor: ResourceUsedFor.Donation,
                translations: [
                    {
                        language: "en",
                        name: "Support the Project",
                    },
                ],
            });

            expect(projectConfig.resources).toHaveLength(1);
            expect(projectConfig.resources[0].usedFor).toBe(ResourceUsedFor.Donation);
        });

        it("should remove resources", () => {
            const projectConfig = new ProjectVersionConfig({
                config: {
                    __version: "1.0",
                    resources: [
                        {
                            link: "https://resource1.com",
                            translations: [{ language: "en", name: "Resource 1" }],
                        },
                        {
                            link: "https://resource2.com",
                            translations: [{ language: "en", name: "Resource 2" }],
                        },
                        {
                            link: "https://resource3.com",
                            translations: [{ language: "en", name: "Resource 3" }],
                        },
                    ],
                },
            });

            projectConfig.removeResource(1); // Remove middle resource

            expect(projectConfig.resources).toHaveLength(2);
            expect(projectConfig.resources[0].link).toBe("https://resource1.com");
            expect(projectConfig.resources[1].link).toBe("https://resource3.com");
        });

        it("should update resources", () => {
            const projectConfig = new ProjectVersionConfig({
                config: {
                    __version: "1.0",
                    resources: [
                        {
                            link: "https://old-docs.com",
                            translations: [{ language: "en", name: "Old Docs" }],
                        },
                    ],
                },
            });

            projectConfig.updateResource(0, {
                link: "https://new-docs.com",
                usedFor: ResourceUsedFor.Tutorial,
                translations: [
                    {
                        language: "en",
                        name: "New Tutorial Docs",
                        description: "Updated documentation",
                    },
                ],
            });

            expect(projectConfig.resources[0].link).toBe("https://new-docs.com");
            expect(projectConfig.resources[0].usedFor).toBe(ResourceUsedFor.Tutorial);
            expect(projectConfig.resources[0].translations[0].name).toBe("New Tutorial Docs");
        });

        it("should get resources by type", () => {
            const projectConfig = new ProjectVersionConfig({
                config: {
                    __version: "1.0",
                    resources: [
                        {
                            link: "https://twitter.com/project",
                            usedFor: ResourceUsedFor.Social,
                            translations: [{ language: "en", name: "Twitter" }],
                        },
                        {
                            link: "https://api.project.com",
                            usedFor: ResourceUsedFor.ExternalService,
                            translations: [{ language: "en", name: "API" }],
                        },
                        {
                            link: "https://facebook.com/project",
                            usedFor: ResourceUsedFor.Social,
                            translations: [{ language: "en", name: "Facebook" }],
                        },
                        {
                            link: "https://calendar.project.com",
                            usedFor: ResourceUsedFor.Scheduling,
                            translations: [{ language: "en", name: "Events Calendar" }],
                        },
                    ],
                },
            });

            const socialResources = projectConfig.getResourcesByType(ResourceUsedFor.Social);

            expect(socialResources).toHaveLength(2);
            expect(socialResources[0].link).toBe("https://twitter.com/project");
            expect(socialResources[1].link).toBe("https://facebook.com/project");

            const schedulingResources = projectConfig.getResourcesByType(ResourceUsedFor.Scheduling);
            expect(schedulingResources).toHaveLength(1);
            expect(schedulingResources[0].link).toBe("https://calendar.project.com");
        });
    });

    describe("Complex scenarios", () => {
        it("should handle projects with comprehensive resource ecosystem", () => {
            const config: ProjectVersionConfigObject = {
                __version: "1.0",
                resources: [
                    // Development resources
                    {
                        link: "https://github.com/org/project",
                        usedFor: ResourceUsedFor.Developer,
                        translations: [
                            { language: "en", name: "GitHub Repository" },
                        ],
                    },
                    {
                        link: "https://ci.project.com",
                        usedFor: ResourceUsedFor.Developer,
                        translations: [
                            { language: "en", name: "CI/CD Pipeline" },
                        ],
                    },
                    // Community resources
                    {
                        link: "https://discord.gg/project",
                        usedFor: ResourceUsedFor.Community,
                        translations: [
                            { language: "en", name: "Discord Server" },
                        ],
                    },
                    {
                        link: "https://forum.project.com",
                        usedFor: ResourceUsedFor.Community,
                        translations: [
                            { language: "en", name: "Community Forum" },
                        ],
                    },
                    // Learning resources
                    {
                        link: "https://learn.project.com",
                        usedFor: ResourceUsedFor.Learning,
                        translations: [
                            { language: "en", name: "Learning Platform" },
                        ],
                    },
                    {
                        link: "https://youtube.com/projectchannel",
                        usedFor: ResourceUsedFor.Tutorial,
                        translations: [
                            { language: "en", name: "YouTube Tutorials" },
                        ],
                    },
                    // Official resources
                    {
                        link: "https://project.com",
                        usedFor: ResourceUsedFor.OfficialWebsite,
                        translations: [
                            { language: "en", name: "Official Website" },
                        ],
                    },
                    // Support resources
                    {
                        link: "https://opencollective.com/project",
                        usedFor: ResourceUsedFor.Donation,
                        translations: [
                            { language: "en", name: "Open Collective" },
                        ],
                    },
                    // Integration resources
                    {
                        link: "https://api.project.com/v2",
                        usedFor: ResourceUsedFor.ExternalService,
                        translations: [
                            { language: "en", name: "REST API v2" },
                        ],
                    },
                    // Proposal resources
                    {
                        link: "https://rfcs.project.com",
                        usedFor: ResourceUsedFor.Proposal,
                        translations: [
                            { language: "en", name: "RFCs & Proposals" },
                        ],
                    },
                ],
            };

            const projectConfig = new ProjectVersionConfig({ config });

            expect(projectConfig.resources).toHaveLength(10);
            expect(projectConfig.getResourcesByType(ResourceUsedFor.Developer)).toHaveLength(2);
            expect(projectConfig.getResourcesByType(ResourceUsedFor.Community)).toHaveLength(2);
            expect(projectConfig.getResourcesByType(ResourceUsedFor.Learning)).toHaveLength(1);
            expect(projectConfig.getResourcesByType(ResourceUsedFor.Tutorial)).toHaveLength(1);
            expect(projectConfig.getResourcesByType(ResourceUsedFor.OfficialWebsite)).toHaveLength(1);
            expect(projectConfig.getResourcesByType(ResourceUsedFor.Donation)).toHaveLength(1);
            expect(projectConfig.getResourcesByType(ResourceUsedFor.ExternalService)).toHaveLength(1);
            expect(projectConfig.getResourcesByType(ResourceUsedFor.Proposal)).toHaveLength(1);
        });

        it("should handle resources with missing optional fields", () => {
            const config: ProjectVersionConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://minimal.com",
                        translations: [
                            {
                                language: "en",
                                name: "Minimal Resource",
                                // No description
                            },
                        ],
                        // No usedFor
                    },
                    {
                        link: "https://complete.com",
                        usedFor: ResourceUsedFor.Feed,
                        translations: [
                            {
                                language: "en",
                                name: "Complete Resource",
                                description: "Has all fields",
                            },
                        ],
                    },
                ],
            };

            const projectConfig = new ProjectVersionConfig({ config });

            expect(projectConfig.resources[0].usedFor).toBeUndefined();
            expect(projectConfig.resources[0].translations[0].description).toBeUndefined();
            expect(projectConfig.resources[1].usedFor).toBe(ResourceUsedFor.Feed);
            expect(projectConfig.resources[1].translations[0].description).toBe("Has all fields");
        });

        it("should handle project evolution through resource updates", () => {
            // Start with basic project
            const projectConfig = new ProjectVersionConfig({
                config: {
                    __version: "1.0",
                    resources: [
                        {
                            link: "https://github.com/user/project",
                            usedFor: ResourceUsedFor.Developer,
                            translations: [{ language: "en", name: "Personal Project" }],
                        },
                    ],
                },
            });

            // Project grows, add community
            projectConfig.addResource({
                link: "https://discord.gg/newproject",
                usedFor: ResourceUsedFor.Community,
                translations: [{ language: "en", name: "Project Discord" }],
            });

            // Add documentation
            projectConfig.addResource({
                link: "https://docs.project.com",
                usedFor: ResourceUsedFor.OfficialWebsite,
                translations: [{ language: "en", name: "Documentation" }],
            });

            // Update original repo (moved to org)
            projectConfig.updateResource(0, {
                link: "https://github.com/org/project",
                translations: [{ language: "en", name: "Organization Project" }],
            });

            // Add funding
            projectConfig.addResource({
                link: "https://patreon.com/project",
                usedFor: ResourceUsedFor.Donation,
                translations: [{ language: "en", name: "Support on Patreon" }],
            });

            const final = projectConfig.export();
            expect(final.resources).toHaveLength(4);
            expect(final.resources[0].link).toBe("https://github.com/org/project");
            expect(final.resources[0].translations[0].name).toBe("Organization Project");
        });

        it("should handle resource categorization for project dashboard", () => {
            const projectConfig = new ProjectVersionConfig({
                config: {
                    __version: "1.0",
                    resources: [
                        // Multiple resources of same type
                        {
                            link: "https://github.com/project/main",
                            usedFor: ResourceUsedFor.Developer,
                            translations: [{ language: "en", name: "Main Repo" }],
                        },
                        {
                            link: "https://github.com/project/plugins",
                            usedFor: ResourceUsedFor.Developer,
                            translations: [{ language: "en", name: "Plugins Repo" }],
                        },
                        {
                            link: "https://github.com/project/examples",
                            usedFor: ResourceUsedFor.Developer,
                            translations: [{ language: "en", name: "Examples" }],
                        },
                        // Social presence
                        {
                            link: "https://twitter.com/project",
                            usedFor: ResourceUsedFor.Social,
                            translations: [{ language: "en", name: "Twitter" }],
                        },
                        {
                            link: "https://linkedin.com/company/project",
                            usedFor: ResourceUsedFor.Social,
                            translations: [{ language: "en", name: "LinkedIn" }],
                        },
                    ],
                },
            });

            // Simulate building a categorized view
            const categories = {
                development: projectConfig.getResourcesByType(ResourceUsedFor.Developer),
                social: projectConfig.getResourcesByType(ResourceUsedFor.Social),
                learning: projectConfig.getResourcesByType(ResourceUsedFor.Learning),
                community: projectConfig.getResourcesByType(ResourceUsedFor.Community),
            };

            expect(categories.development).toHaveLength(3);
            expect(categories.social).toHaveLength(2);
            expect(categories.learning).toHaveLength(0);
            expect(categories.community).toHaveLength(0);
        });

        it("should maintain data integrity through serialization cycle", () => {
            const originalConfig: ProjectVersionConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://test.com/resource",
                        usedFor: ResourceUsedFor.Install,
                        translations: [
                            {
                                language: "en",
                                name: "Installation Guide",
                                description: "Step-by-step installation instructions",
                            },
                            {
                                language: "zh",
                                name: "安装指南",
                                description: "分步安装说明",
                            },
                        ],
                    },
                ],
            };

            // Create, export, re-import cycle
            const config1 = new ProjectVersionConfig({ config: originalConfig });
            const exported1 = config1.export();
            const config2 = new ProjectVersionConfig({ config: exported1 });
            const exported2 = config2.export();

            // Should remain identical
            expect(exported2).toEqual(originalConfig);
            expect(exported2.resources[0].translations).toHaveLength(2);
        });
    });
});