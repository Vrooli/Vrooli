import { describe, it, expect, vi, beforeEach } from "vitest";
import { NoteVersionConfig, type NoteVersionConfigObject } from "./note.js";
import { type ResourceVersion } from "../../api/types.js";
import { ResourceUsedFor } from "./base.js";

describe("NoteVersionConfig", () => {
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
        it("should create NoteVersionConfig with complete data", () => {
            const config: NoteVersionConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://example.com/note-reference",
                        usedFor: ResourceUsedFor.Notes,
                        translations: [
                            {
                                language: "en",
                                name: "Reference Material",
                                description: "Additional reference for this note",
                            },
                        ],
                    },
                    {
                        link: "https://docs.example.com/tutorial",
                        usedFor: ResourceUsedFor.Tutorial,
                        translations: [
                            {
                                language: "en",
                                name: "Related Tutorial",
                            },
                        ],
                    },
                ],
            };

            const noteConfig = new NoteVersionConfig({ config });

            expect(noteConfig.__version).toBe("1.0");
            expect(noteConfig.resources).toHaveLength(2);
            expect(noteConfig.resources[0].link).toBe("https://example.com/note-reference");
            expect(noteConfig.resources[0].usedFor).toBe(ResourceUsedFor.Notes);
            expect(noteConfig.resources[1].usedFor).toBe(ResourceUsedFor.Tutorial);
        });

        it("should create NoteVersionConfig with minimal data", () => {
            const config: NoteVersionConfigObject = {
                __version: "1.0",
            };

            const noteConfig = new NoteVersionConfig({ config });

            expect(noteConfig.__version).toBe("1.0");
            expect(noteConfig.resources).toEqual([]);
        });

        it("should create NoteVersionConfig without version (uses default)", () => {
            const config: Partial<NoteVersionConfigObject> = {
                resources: [],
            };

            const noteConfig = new NoteVersionConfig({ config: config as NoteVersionConfigObject });

            expect(noteConfig.__version).toBe("1.0");
            expect(noteConfig.resources).toEqual([]);
        });
    });

    describe("parse", () => {
        it("should parse valid note version data", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: {
                    __version: "1.0",
                    resources: [
                        {
                            link: "https://example.com/resource",
                            translations: [
                                {
                                    language: "en",
                                    name: "Resource",
                                },
                            ],
                        },
                    ],
                },
            };

            const noteConfig = NoteVersionConfig.parse(versionData, mockLogger);

            expect(noteConfig.__version).toBe("1.0");
            expect(noteConfig.resources).toHaveLength(1);
            expect(noteConfig.resources[0].link).toBe("https://example.com/resource");
        });

        it("should parse with null config", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: null as any,
            };

            const noteConfig = NoteVersionConfig.parse(versionData, mockLogger);

            expect(noteConfig.__version).toBe("1.0");
            expect(noteConfig.resources).toEqual([]);
        });

        it("should parse with undefined config", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: undefined as any,
            };

            const noteConfig = NoteVersionConfig.parse(versionData, mockLogger);

            expect(noteConfig.__version).toBe("1.0");
            expect(noteConfig.resources).toEqual([]);
        });

        it("should parse with useFallbacks parameter (no effect for NoteVersionConfig)", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: {
                    __version: "1.0",
                },
            };

            const configWithFallbacks = NoteVersionConfig.parse(versionData, mockLogger, { useFallbacks: true });
            const configWithoutFallbacks = NoteVersionConfig.parse(versionData, mockLogger, { useFallbacks: false });

            // Both should be the same as NoteVersionConfig doesn't have additional fallback properties
            expect(configWithFallbacks.export()).toEqual(configWithoutFallbacks.export());
        });
    });

    describe("default", () => {
        it("should create default NoteVersionConfig", () => {
            const noteConfig = NoteVersionConfig.default();

            expect(noteConfig.__version).toBe("1.0");
            expect(noteConfig.resources).toEqual([]);
        });
    });

    describe("export", () => {
        it("should export all properties", () => {
            const originalConfig: NoteVersionConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://research.example.com",
                        usedFor: ResourceUsedFor.Researching,
                        translations: [
                            {
                                language: "en",
                                name: "Research Paper",
                                description: "Academic research related to this note",
                            },
                            {
                                language: "es",
                                name: "Documento de Investigación",
                                description: "Investigación académica relacionada con esta nota",
                            },
                        ],
                    },
                ],
            };

            const noteConfig = new NoteVersionConfig({ config: originalConfig });
            const exported = noteConfig.export();

            expect(exported).toEqual(originalConfig);
        });

        it("should export minimal config", () => {
            const noteConfig = new NoteVersionConfig({ config: { __version: "1.0" } });
            const exported = noteConfig.export();

            expect(exported).toEqual({
                __version: "1.0",
                resources: [],
            });
        });
    });

    describe("Resource management (inherited from BaseConfig)", () => {
        it("should add resources", () => {
            const noteConfig = NoteVersionConfig.default();

            noteConfig.addResource({
                link: "https://example.com/reference",
                usedFor: ResourceUsedFor.Related,
                translations: [
                    {
                        language: "en",
                        name: "Related Content",
                    },
                ],
            });

            expect(noteConfig.resources).toHaveLength(1);
            expect(noteConfig.resources[0].usedFor).toBe(ResourceUsedFor.Related);
        });

        it("should remove resources", () => {
            const noteConfig = new NoteVersionConfig({
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
                        {
                            link: "https://example3.com",
                            translations: [{ language: "en", name: "Resource 3" }],
                        },
                    ],
                },
            });

            noteConfig.removeResource(1); // Remove middle resource

            expect(noteConfig.resources).toHaveLength(2);
            expect(noteConfig.resources[0].link).toBe("https://example1.com");
            expect(noteConfig.resources[1].link).toBe("https://example3.com");
        });

        it("should update resources", () => {
            const noteConfig = new NoteVersionConfig({
                config: {
                    __version: "1.0",
                    resources: [
                        {
                            link: "https://old-link.com",
                            translations: [{ language: "en", name: "Old Name" }],
                        },
                    ],
                },
            });

            noteConfig.updateResource(0, {
                link: "https://new-link.com",
                usedFor: ResourceUsedFor.Learning,
                translations: [
                    {
                        language: "en",
                        name: "Updated Name",
                        description: "New description",
                    },
                ],
            });

            expect(noteConfig.resources[0].link).toBe("https://new-link.com");
            expect(noteConfig.resources[0].usedFor).toBe(ResourceUsedFor.Learning);
            expect(noteConfig.resources[0].translations[0].name).toBe("Updated Name");
        });

        it("should get resources by type", () => {
            const noteConfig = new NoteVersionConfig({
                config: {
                    __version: "1.0",
                    resources: [
                        {
                            link: "https://learn1.com",
                            usedFor: ResourceUsedFor.Learning,
                            translations: [{ language: "en", name: "Learning 1" }],
                        },
                        {
                            link: "https://context.com",
                            usedFor: ResourceUsedFor.Context,
                            translations: [{ language: "en", name: "Context" }],
                        },
                        {
                            link: "https://learn2.com",
                            usedFor: ResourceUsedFor.Learning,
                            translations: [{ language: "en", name: "Learning 2" }],
                        },
                        {
                            link: "https://tutorial.com",
                            usedFor: ResourceUsedFor.Tutorial,
                            translations: [{ language: "en", name: "Tutorial" }],
                        },
                    ],
                },
            });

            const learningResources = noteConfig.getResourcesByType(ResourceUsedFor.Learning);

            expect(learningResources).toHaveLength(2);
            expect(learningResources[0].link).toBe("https://learn1.com");
            expect(learningResources[1].link).toBe("https://learn2.com");

            const contextResources = noteConfig.getResourcesByType(ResourceUsedFor.Context);
            expect(contextResources).toHaveLength(1);
            expect(contextResources[0].link).toBe("https://context.com");
        });
    });

    describe("Complex scenarios", () => {
        it("should handle notes with diverse resource types", () => {
            const config: NoteVersionConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://github.com/example/code-snippets",
                        usedFor: ResourceUsedFor.Developer,
                        translations: [
                            { language: "en", name: "Code Examples" },
                        ],
                    },
                    {
                        link: "https://research-paper.pdf",
                        usedFor: ResourceUsedFor.Researching,
                        translations: [
                            { language: "en", name: "Research Paper", description: "Academic research on the topic" },
                        ],
                    },
                    {
                        link: "https://youtube.com/watch?v=tutorial",
                        usedFor: ResourceUsedFor.Tutorial,
                        translations: [
                            { language: "en", name: "Video Tutorial" },
                            { language: "fr", name: "Tutoriel Vidéo" },
                        ],
                    },
                    {
                        link: "https://related-article.com",
                        usedFor: ResourceUsedFor.Related,
                        translations: [
                            { language: "en", name: "Related Article" },
                        ],
                    },
                    {
                        link: "https://context-doc.pdf",
                        usedFor: ResourceUsedFor.Context,
                        translations: [
                            { language: "en", name: "Background Information" },
                        ],
                    },
                ],
            };

            const noteConfig = new NoteVersionConfig({ config });

            expect(noteConfig.resources).toHaveLength(5);
            expect(noteConfig.getResourcesByType(ResourceUsedFor.Developer)).toHaveLength(1);
            expect(noteConfig.getResourcesByType(ResourceUsedFor.Researching)).toHaveLength(1);
            expect(noteConfig.getResourcesByType(ResourceUsedFor.Tutorial)).toHaveLength(1);
            expect(noteConfig.getResourcesByType(ResourceUsedFor.Related)).toHaveLength(1);
            expect(noteConfig.getResourcesByType(ResourceUsedFor.Context)).toHaveLength(1);
        });

        it("should handle multilingual resources", () => {
            const config: NoteVersionConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://multilingual-resource.com",
                        usedFor: ResourceUsedFor.Learning,
                        translations: [
                            {
                                language: "en",
                                name: "Learning Resource",
                                description: "A comprehensive guide",
                            },
                            {
                                language: "es",
                                name: "Recurso de Aprendizaje",
                                description: "Una guía completa",
                            },
                            {
                                language: "fr",
                                name: "Ressource d'Apprentissage",
                                description: "Un guide complet",
                            },
                            {
                                language: "de",
                                name: "Lernressource",
                                description: "Ein umfassender Leitfaden",
                            },
                        ],
                    },
                ],
            };

            const noteConfig = new NoteVersionConfig({ config });

            expect(noteConfig.resources[0].translations).toHaveLength(4);
            expect(noteConfig.resources[0].translations.map(t => t.language)).toEqual(["en", "es", "fr", "de"]);
        });

        it("should handle resource lifecycle (add, update, remove)", () => {
            const noteConfig = NoteVersionConfig.default();

            // Add initial resources
            noteConfig.addResource({
                link: "https://resource1.com",
                translations: [{ language: "en", name: "Resource 1" }],
            });
            noteConfig.addResource({
                link: "https://resource2.com",
                translations: [{ language: "en", name: "Resource 2" }],
            });
            noteConfig.addResource({
                link: "https://resource3.com",
                translations: [{ language: "en", name: "Resource 3" }],
            });

            expect(noteConfig.resources).toHaveLength(3);

            // Update middle resource
            noteConfig.updateResource(1, {
                usedFor: ResourceUsedFor.Notes,
                translations: [
                    { language: "en", name: "Updated Resource 2", description: "Now with description" },
                ],
            });

            expect(noteConfig.resources[1].usedFor).toBe(ResourceUsedFor.Notes);
            expect(noteConfig.resources[1].translations[0].description).toBe("Now with description");

            // Remove first resource
            noteConfig.removeResource(0);

            expect(noteConfig.resources).toHaveLength(2);
            expect(noteConfig.resources[0].link).toBe("https://resource2.com"); // Previously index 1
            expect(noteConfig.resources[1].link).toBe("https://resource3.com"); // Previously index 2

            // Add another resource
            noteConfig.addResource({
                link: "https://resource4.com",
                usedFor: ResourceUsedFor.Feed,
                translations: [{ language: "en", name: "Resource 4" }],
            });

            expect(noteConfig.resources).toHaveLength(3);
            expect(noteConfig.resources[2].link).toBe("https://resource4.com");
        });

        it("should maintain data integrity through export/import cycle", () => {
            const originalConfig: NoteVersionConfigObject = {
                __version: "1.0",
                resources: [
                    {
                        link: "https://example.com/resource",
                        usedFor: ResourceUsedFor.ExternalService,
                        translations: [
                            {
                                language: "en",
                                name: "External API",
                                description: "Integration with external service",
                            },
                        ],
                    },
                ],
            };

            // Create config, export it
            const noteConfig1 = new NoteVersionConfig({ config: originalConfig });
            const exported = noteConfig1.export();

            // Create new config from exported data
            const noteConfig2 = new NoteVersionConfig({ config: exported });
            const reExported = noteConfig2.export();

            // Should be identical
            expect(reExported).toEqual(originalConfig);
        });
    });
});