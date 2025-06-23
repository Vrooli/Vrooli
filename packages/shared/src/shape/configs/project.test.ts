import { describe, it, expect, vi, beforeEach } from "vitest";
import { ProjectVersionConfig, type ProjectVersionConfigObject } from "./project.js";
import { type ResourceVersion } from "../../api/types.js";
import { ResourceUsedFor } from "./base.js";
import { projectConfigFixtures } from "../../__test/fixtures/config/projectConfigFixtures.js";
import { runComprehensiveConfigTests } from "./__test/configTestUtils.js";

describe("ProjectVersionConfig", () => {
    // Standardized config tests using fixtures
    runComprehensiveConfigTests(
        ProjectVersionConfig,
        projectConfigFixtures,
        "project",
    );

    // Project-specific business logic tests
    describe("project-specific functionality", () => {
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
            const config = projectConfigFixtures.complete;

            const projectConfig = new ProjectVersionConfig({ config });

            expect(projectConfig.__version).toBe(config.__version);
            expect(projectConfig.resources).toHaveLength(3);
            expect(projectConfig.resources[0].link).toBe("https://example.com/project-docs");
            expect(projectConfig.resources[0].usedFor).toBe(ResourceUsedFor.OfficialWebsite);
            expect(projectConfig.resources[1].usedFor).toBe(ResourceUsedFor.Developer);
        });

        it("should create ProjectVersionConfig with minimal data", () => {
            const config = projectConfigFixtures.minimal;

            const projectConfig = new ProjectVersionConfig({ config });

            expect(projectConfig.__version).toBe(config.__version);
            expect(projectConfig.resources).toEqual([]);
        });

        it("should create ProjectVersionConfig without version (uses default)", () => {
            const config = projectConfigFixtures.withDefaults;

            const projectConfig = new ProjectVersionConfig({ config });

            expect(projectConfig.__version).toBe(config.__version);
            expect(projectConfig.resources).toEqual([]);
        });
    });

    describe("parse", () => {
        it("should parse valid project version data", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: projectConfigFixtures.variants.minimalWithResource,
            };

            const projectConfig = ProjectVersionConfig.parse(versionData, mockLogger);

            expect(projectConfig.__version).toBe(projectConfigFixtures.variants.minimalWithResource.__version);
            expect(projectConfig.resources).toHaveLength(1);
            expect(projectConfig.resources[0].link).toBe("https://project.example.com");
        });

        it("should parse with null config", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: null as any,
            };

            const projectConfig = ProjectVersionConfig.parse(versionData, mockLogger);

            expect(projectConfig.__version).toBe(projectConfigFixtures.minimal.__version);
            expect(projectConfig.resources).toEqual([]);
        });

        it("should parse with undefined config", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: undefined as any,
            };

            const projectConfig = ProjectVersionConfig.parse(versionData, mockLogger);

            expect(projectConfig.__version).toBe(projectConfigFixtures.minimal.__version);
            expect(projectConfig.resources).toEqual([]);
        });

        it("should parse with useFallbacks parameter (no effect for ProjectVersionConfig)", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: projectConfigFixtures.minimal,
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

            expect(projectConfig.__version).toBe(projectConfigFixtures.minimal.__version);
            expect(projectConfig.resources).toEqual([]);
        });
    });

    describe("export", () => {
        it("should export all properties", () => {
            const originalConfig = projectConfigFixtures.variants.multiLanguageProject;

            const projectConfig = new ProjectVersionConfig({ config: originalConfig });
            const exported = projectConfig.export();

            expect(exported).toEqual(originalConfig);
        });

        it("should export minimal config", () => {
            const projectConfig = new ProjectVersionConfig({ config: projectConfigFixtures.minimal });
            const exported = projectConfig.export();

            expect(exported).toEqual({
                __version: projectConfigFixtures.minimal.__version,
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
            const config = projectConfigFixtures.variants.openSourceProject;

            const projectConfig = new ProjectVersionConfig({ config });

            expect(projectConfig.resources).toHaveLength(4);
            expect(projectConfig.getResourcesByType(ResourceUsedFor.Developer)).toHaveLength(1);
            expect(projectConfig.getResourcesByType(ResourceUsedFor.Community)).toHaveLength(1);
            expect(projectConfig.getResourcesByType(ResourceUsedFor.OfficialWebsite)).toHaveLength(1);
            expect(projectConfig.getResourcesByType(ResourceUsedFor.Donation)).toHaveLength(1);
        });

        it("should handle resources with missing optional fields", () => {
            const config = projectConfigFixtures.variants.socialProject;

            const projectConfig = new ProjectVersionConfig({ config });

            expect(projectConfig.resources[0].usedFor).toBe(ResourceUsedFor.Social);
            expect(projectConfig.resources[0].translations[0].description).toBe("Follow us for updates");
            expect(projectConfig.resources[2].usedFor).toBe(ResourceUsedFor.Feed);
            expect(projectConfig.resources[2].translations[0].description).toBe("Project blog and announcements");
        });

        it("should handle project evolution through resource updates", () => {
            // Start with basic project
            const projectConfig = new ProjectVersionConfig({
                config: projectConfigFixtures.variants.minimalWithResource,
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
                config: projectConfigFixtures.variants.socialProject,
            });

            // Simulate building a categorized view
            const categories = {
                development: projectConfig.getResourcesByType(ResourceUsedFor.Developer),
                social: projectConfig.getResourcesByType(ResourceUsedFor.Social),
                learning: projectConfig.getResourcesByType(ResourceUsedFor.Learning),
                community: projectConfig.getResourcesByType(ResourceUsedFor.Community),
            };

            expect(categories.development).toHaveLength(0);
            expect(categories.social).toHaveLength(2);
            expect(categories.learning).toHaveLength(0);
            expect(categories.community).toHaveLength(0);
        });

        it("should maintain data integrity through serialization cycle", () => {
            const originalConfig = projectConfigFixtures.variants.multiLanguageProject;

            // Create, export, re-import cycle
            const config1 = new ProjectVersionConfig({ config: originalConfig });
            const exported1 = config1.export();
            const config2 = new ProjectVersionConfig({ config: exported1 });
            const exported2 = config2.export();

            // Should remain identical
            expect(exported2).toEqual(originalConfig);
            expect(exported2.resources[0].translations).toHaveLength(5);
        });
    });
    });
});
