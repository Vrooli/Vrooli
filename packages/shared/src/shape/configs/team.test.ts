import { describe, it, expect, vi, beforeEach } from "vitest";
import { TeamConfig, type TeamConfigObject } from "./team.js";
import { type Team } from "../../api/types.js";
import { ResourceUsedFor } from "./base.js";
import { teamConfigFixtures } from "../../__test/fixtures/config/teamConfigFixtures.js";
import { runComprehensiveConfigTests } from "./__test/configTestUtils.js";

describe("TeamConfig", () => {
    // Standardized config tests using fixtures
    runComprehensiveConfigTests(
        TeamConfig,
        teamConfigFixtures,
        "team",
    );

    // Team-specific business logic tests
    describe("team-specific functionality", () => {
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
        it("should create TeamConfig with complete data", () => {
            const config: TeamConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://example.com/team-docs",
                        usedFor: ResourceUsedFor.OfficialWebsite,
                        translations: [
                            {
                                language: "en",
                                name: "Team Documentation",
                                description: "Official team documentation",
                            },
                        ],
                    },
                ],
                structure: {
                    type: "MOISE+",
                    version: "1.0",
                    content: `
                        structure Swarm {
                            group devTeam {
                                role leader cardinality 1..1
                                role developer cardinality 2..5
                                role tester cardinality 1..2
                                link leader > developer
                                link leader > tester
                            }
                        }
                    `,
                },
            };

            const teamConfig = new TeamConfig({ config });

            expect(teamConfig.__version).toBe("1.0");
            expect(teamConfig.resources).toHaveLength(1);
            expect(teamConfig.resources[0].link).toBe("https://example.com/team-docs");
            expect(teamConfig.structure?.type).toBe("MOISE+");
            expect(teamConfig.structure?.version).toBe("1.0");
            expect(teamConfig.structure?.content).toContain("structure Swarm");
        });

        it("should create TeamConfig with minimal data", () => {
            const config: TeamConfigObject = {
                __version: "1.0",
            };

            const teamConfig = new TeamConfig({ config });

            expect(teamConfig.__version).toBe("1.0");
            expect(teamConfig.resources).toEqual([]);
            expect(teamConfig.structure).toBeUndefined();
        });

        it("should create TeamConfig with only resources", () => {
            const config: TeamConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://github.com/team/repo",
                        usedFor: ResourceUsedFor.Developer,
                        translations: [
                            {
                                language: "en",
                                name: "Team Repository",
                            },
                        ],
                    },
                    {
                        link: "https://discord.gg/team",
                        usedFor: ResourceUsedFor.Community,
                        translations: [
                            {
                                language: "en",
                                name: "Team Discord",
                                description: "Join our community",
                            },
                        ],
                    },
                ],
            };

            const teamConfig = new TeamConfig({ config });

            expect(teamConfig.resources).toHaveLength(2);
            expect(teamConfig.resources[0].usedFor).toBe(ResourceUsedFor.Developer);
            expect(teamConfig.resources[1].usedFor).toBe(ResourceUsedFor.Community);
        });
    });

    describe("parse", () => {
        it("should parse with useFallbacks true (default)", () => {
            const teamData: Pick<Team, "config"> = {
                config: {
                    __version: "1.0",
                    resources: [],
                },
            };

            const teamConfig = TeamConfig.parse(teamData, mockLogger);

            expect(teamConfig.__version).toBe("1.0");
            expect(teamConfig.resources).toEqual([]);
            expect(teamConfig.structure).toEqual({
                type: "MOISE+",
                version: "1.0",
                content: "",
            });
        });

        it("should parse with useFallbacks false", () => {
            const teamData: Pick<Team, "config"> = {
                config: {
                    __version: "1.0",
                    resources: [],
                },
            };

            const teamConfig = TeamConfig.parse(teamData, mockLogger, { useFallbacks: false });

            expect(teamConfig.__version).toBe("1.0");
            expect(teamConfig.resources).toEqual([]);
            expect(teamConfig.structure).toBeUndefined();
        });

        it("should parse with existing structure", () => {
            const teamData: Pick<Team, "config"> = {
                config: {
                    __version: "1.0",
                    resources: [],
                    structure: {
                        type: "FIPA ACL",
                        version: "2.0",
                        content: "FIPA ACL content here",
                    },
                },
            };

            const teamConfig = TeamConfig.parse(teamData, mockLogger);

            expect(teamConfig.structure?.type).toBe("FIPA ACL");
            expect(teamConfig.structure?.version).toBe("2.0");
            expect(teamConfig.structure?.content).toBe("FIPA ACL content here");
        });

        it("should handle null config", () => {
            const teamData: Pick<Team, "config"> = {
                config: null as any,
            };

            const teamConfig = TeamConfig.parse(teamData, mockLogger);

            expect(teamConfig.__version).toBe("1.0");
            expect(teamConfig.resources).toEqual([]);
            expect(teamConfig.structure).toEqual({
                type: "MOISE+",
                version: "1.0",
                content: "",
            });
        });

        it("should handle undefined config", () => {
            const teamData: Pick<Team, "config"> = {
                config: undefined as any,
            };

            const teamConfig = TeamConfig.parse(teamData, mockLogger);

            expect(teamConfig.__version).toBe("1.0");
            expect(teamConfig.resources).toEqual([]);
            expect(teamConfig.structure).toEqual({
                type: "MOISE+",
                version: "1.0",
                content: "",
            });
        });
    });

    describe("export", () => {
        it("should export all properties", () => {
            const originalConfig: TeamConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://example.com",
                        usedFor: ResourceUsedFor.Learning,
                        translations: [
                            {
                                language: "en",
                                name: "Learning Resource",
                            },
                        ],
                    },
                ],
                structure: {
                    type: "ArchiMate",
                    version: "3.1",
                    content: "<archimate>...</archimate>",
                },
            };

            const teamConfig = new TeamConfig({ config: originalConfig });
            const exported = teamConfig.export();

            expect(exported).toEqual(originalConfig);
        });

        it("should export minimal config", () => {
            const teamConfig = new TeamConfig({ config: { __version: "1.0" } });
            const exported = teamConfig.export();

            expect(exported).toEqual({
                __version: "1.0",
                resources: [],
                structure: undefined,
            });
        });
    });

    describe("setStructure", () => {
        it("should set new structure", () => {
            const teamConfig = new TeamConfig({ config: { __version: "1.0" } });
            
            const newStructure: TeamConfigObject["structure"] = {
                type: "MOISE+",
                version: "1.0",
                content: "structure TestOrg { }",
            };

            teamConfig.setStructure(newStructure);

            expect(teamConfig.structure).toEqual(newStructure);
        });

        it("should replace existing structure", () => {
            const teamConfig = new TeamConfig({ 
                config: { 
                    __version: "1.0",
                    structure: {
                        type: "Old",
                        content: "old content",
                    },
                },
            });
            
            const newStructure: TeamConfigObject["structure"] = {
                type: "MOISE+",
                version: "2.0",
                content: "new content",
            };

            teamConfig.setStructure(newStructure);

            expect(teamConfig.structure).toEqual(newStructure);
        });

        it("should allow setting undefined structure", () => {
            const teamConfig = new TeamConfig({ 
                config: { 
                    __version: "1.0",
                    structure: {
                        type: "MOISE+",
                        content: "content",
                    },
                },
            });

            teamConfig.setStructure(undefined);

            expect(teamConfig.structure).toBeUndefined();
        });
    });

    describe("Resource management (inherited from BaseConfig)", () => {
        it("should add resources", () => {
            const teamConfig = new TeamConfig({ config: { __version: "1.0" } });

            teamConfig.addResource({
                link: "https://example.com/resource",
                usedFor: ResourceUsedFor.Tutorial,
                translations: [
                    {
                        language: "en",
                        name: "Tutorial",
                    },
                ],
            });

            expect(teamConfig.resources).toHaveLength(1);
            expect(teamConfig.resources[0].usedFor).toBe(ResourceUsedFor.Tutorial);
        });

        it("should remove resources", () => {
            const teamConfig = new TeamConfig({ 
                config: { 
                    __version: "1.0",
                    resources: [
                        {
                            link: "https://example1.com",
                            translations: [{ language: "en", name: "Resource 1" }],
                        },
                        {
                            link: "https://example2.com",
                            translations: [{ language: "en", name: "Resource 2" }],
                        },
                    ],
                },
            });

            teamConfig.removeResource(0);

            expect(teamConfig.resources).toHaveLength(1);
            expect(teamConfig.resources[0].link).toBe("https://example2.com");
        });

        it("should update resources", () => {
            const teamConfig = new TeamConfig({ 
                config: { 
                    __version: "1.0",
                    resources: [
                        {
                            link: "https://old.com",
                            translations: [{ language: "en", name: "Old Name" }],
                        },
                    ],
                },
            });

            teamConfig.updateResource(0, {
                link: "https://new.com",
                usedFor: ResourceUsedFor.Social,
            });

            expect(teamConfig.resources[0].link).toBe("https://new.com");
            expect(teamConfig.resources[0].usedFor).toBe(ResourceUsedFor.Social);
            expect(teamConfig.resources[0].translations[0].name).toBe("Old Name");
        });

        it("should get resources by type", () => {
            const teamConfig = new TeamConfig({ 
                config: { 
                    __version: "1.0",
                    resources: [
                        {
                            link: "https://social1.com",
                            usedFor: ResourceUsedFor.Social,
                            translations: [{ language: "en", name: "Social 1" }],
                        },
                        {
                            link: "https://dev.com",
                            usedFor: ResourceUsedFor.Developer,
                            translations: [{ language: "en", name: "Dev" }],
                        },
                        {
                            link: "https://social2.com",
                            usedFor: ResourceUsedFor.Social,
                            translations: [{ language: "en", name: "Social 2" }],
                        },
                    ],
                },
            });

            const socialResources = teamConfig.getResourcesByType(ResourceUsedFor.Social);

            expect(socialResources).toHaveLength(2);
            expect(socialResources[0].link).toBe("https://social1.com");
            expect(socialResources[1].link).toBe("https://social2.com");
        });
    });

    describe("Complex organizational structures", () => {
        it("should handle complex MOISE+ structure", () => {
            const complexStructure = {
                type: "MOISE+",
                version: "1.0",
                content: `
                    structure VrooliOrganization {
                        group Platform {
                            role admin cardinality 1..3
                            role moderator cardinality 2..10
                            role developer cardinality 5..20
                            
                            group SecurityTeam {
                                role securityLead cardinality 1..1
                                role securityAnalyst cardinality 2..5
                                link securityLead > securityAnalyst
                            }
                            
                            group AITeam {
                                role aiLead cardinality 1..1
                                role mlEngineer cardinality 3..8
                                role dataScientist cardinality 2..6
                                link aiLead > mlEngineer
                                link aiLead > dataScientist
                            }
                            
                            link admin > moderator
                            link admin > developer
                            link admin > securityLead
                            link admin > aiLead
                        }
                        
                        scheme MainScheme {
                            goal maintainPlatform {
                                goal monitorSecurity
                                goal developFeatures
                                goal trainModels
                            }
                            
                            mission platformMaintenance {
                                goal maintainPlatform
                            }
                        }
                        
                        norm n1 {
                            condition: #goals(maintainPlatform) > 0
                            role: admin
                            type: obligation
                        }
                    }
                `,
            };

            const teamConfig = new TeamConfig({ 
                config: { 
                    __version: "1.0",
                    structure: complexStructure,
                },
            });

            expect(teamConfig.structure?.content).toContain("group Platform");
            expect(teamConfig.structure?.content).toContain("role admin cardinality 1..3");
            expect(teamConfig.structure?.content).toContain("scheme MainScheme");
            expect(teamConfig.structure?.content).toContain("norm n1");
        });

        it("should handle different organizational model types", () => {
            const models = [
                {
                    type: "FIPA ACL",
                    version: "2.0",
                    content: "@prefix fipa: <http://www.fipa.org/schemas/fipa-acl#> .",
                },
                {
                    type: "ArchiMate",
                    version: "3.1",
                    content: "<archimate><elements>...</elements></archimate>",
                },
                {
                    type: "Custom",
                    content: "Custom organizational structure definition",
                },
            ];

            for (const model of models) {
                const teamConfig = new TeamConfig({ 
                    config: { 
                        __version: "1.0",
                        structure: model,
                    },
                });

                expect(teamConfig.structure?.type).toBe(model.type);
                expect(teamConfig.structure?.version).toBe(model.version);
                expect(teamConfig.structure?.content).toBe(model.content);
            }
        });

        it("should handle team with multiple resource types", () => {
            const config: TeamConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://github.com/team/repo",
                        usedFor: ResourceUsedFor.Developer,
                        translations: [
                            { language: "en", name: "Source Code" },
                            { language: "es", name: "Código Fuente" },
                        ],
                    },
                    {
                        link: "https://team.slack.com",
                        usedFor: ResourceUsedFor.Community,
                        translations: [
                            { language: "en", name: "Team Slack", description: "Internal communication" },
                        ],
                    },
                    {
                        link: "https://docs.team.com",
                        usedFor: ResourceUsedFor.Learning,
                        translations: [
                            { language: "en", name: "Documentation", description: "Learn how to use our platform" },
                        ],
                    },
                    {
                        link: "https://calendar.team.com",
                        usedFor: ResourceUsedFor.Scheduling,
                        translations: [
                            { language: "en", name: "Team Calendar" },
                        ],
                    },
                    {
                        link: "https://donate.team.com",
                        usedFor: ResourceUsedFor.Donation,
                        translations: [
                            { language: "en", name: "Support Us", description: "Help fund our mission" },
                        ],
                    },
                ],
                structure: {
                    type: "MOISE+",
                    version: "1.0",
                    content: "structure TeamStructure { }",
                },
            };

            const teamConfig = new TeamConfig({ config });

            expect(teamConfig.resources).toHaveLength(5);
            expect(teamConfig.getResourcesByType(ResourceUsedFor.Developer)).toHaveLength(1);
            expect(teamConfig.getResourcesByType(ResourceUsedFor.Community)).toHaveLength(1);
            expect(teamConfig.getResourcesByType(ResourceUsedFor.Learning)).toHaveLength(1);
            expect(teamConfig.getResourcesByType(ResourceUsedFor.Scheduling)).toHaveLength(1);
            expect(teamConfig.getResourcesByType(ResourceUsedFor.Donation)).toHaveLength(1);
        });
    });
    });
});
