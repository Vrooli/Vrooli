import { describe, it, expect, vi, beforeEach } from "vitest";
import { NoteVersionConfig, type NoteVersionConfigObject } from "./note.js";
import { type ResourceVersion } from "../../api/types.js";
import { ResourceUsedFor } from "./base.js";
import { noteConfigFixtures } from "../../__test/fixtures/config/noteConfigFixtures.js";

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
            const config = noteConfigFixtures.complete;

            const noteConfig = new NoteVersionConfig({ config });

            expect(noteConfig.__version).toBe(config.__version);
            expect(noteConfig.resources).toHaveLength(3);
            expect(noteConfig.resources[0].link).toBe("https://example.com/note-docs");
            expect(noteConfig.resources[0].usedFor).toBe(ResourceUsedFor.Notes);
            expect(noteConfig.resources[1].usedFor).toBe(ResourceUsedFor.Related);
            expect(noteConfig.resources[2].usedFor).toBe(ResourceUsedFor.Developer);
        });

        it("should create NoteVersionConfig with minimal data", () => {
            const config = noteConfigFixtures.minimal;

            const noteConfig = new NoteVersionConfig({ config });

            expect(noteConfig.__version).toBe(config.__version);
            expect(noteConfig.resources).toEqual([]);
        });

        it("should create NoteVersionConfig without version (uses default)", () => {
            const config = noteConfigFixtures.withDefaults as Partial<NoteVersionConfigObject>;
            delete (config as any).__version; // Remove version to test default behavior

            const noteConfig = new NoteVersionConfig({ config: config as NoteVersionConfigObject });

            expect(noteConfig.__version).toBe("1.0");
            expect(noteConfig.resources).toEqual([]);
        });
    });

    describe("parse", () => {
        it("should parse valid note version data", () => {
            const versionData: Pick<ResourceVersion, "config"> = {
                config: noteConfigFixtures.variants.minimalResourceNote,
            };

            const noteConfig = NoteVersionConfig.parse(versionData, mockLogger);

            expect(noteConfig.__version).toBe(versionData.config.__version);
            expect(noteConfig.resources).toHaveLength(1);
            expect(noteConfig.resources[0].link).toBe("https://example.com/note");
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
                config: noteConfigFixtures.minimal,
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
            const originalConfig = noteConfigFixtures.variants.researchNote;

            const noteConfig = new NoteVersionConfig({ config: originalConfig });
            const exported = noteConfig.export();

            expect(exported).toEqual(originalConfig);
        });

        it("should export minimal config", () => {
            const noteConfig = new NoteVersionConfig({ config: noteConfigFixtures.minimal });
            const exported = noteConfig.export();

            expect(exported).toEqual({
                __version: noteConfigFixtures.minimal.__version,
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
                config: noteConfigFixtures.complete,
            });

            noteConfig.removeResource(1); // Remove middle resource

            expect(noteConfig.resources).toHaveLength(2);
            expect(noteConfig.resources[0].link).toBe("https://example.com/note-docs");
            expect(noteConfig.resources[1].link).toBe("https://github.com/example/note-project");
        });

        it("should update resources", () => {
            const noteConfig = new NoteVersionConfig({
                config: noteConfigFixtures.variants.minimalResourceNote,
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
                config: noteConfigFixtures.variants.tutorialNote,
            });

            const learningResources = noteConfig.getResourcesByType(ResourceUsedFor.Learning);

            expect(learningResources).toHaveLength(1);
            expect(learningResources[0].link).toBe("https://example.com/tutorial");

            const tutorialResources = noteConfig.getResourcesByType(ResourceUsedFor.Tutorial);
            expect(tutorialResources).toHaveLength(1);
            expect(tutorialResources[0].link).toBe("https://youtube.com/watch?v=abc123");
        });
    });

    describe("Complex scenarios", () => {
        it("should handle notes with diverse resource types", () => {
            const config = noteConfigFixtures.variants.communityNote;

            const noteConfig = new NoteVersionConfig({ config });

            expect(noteConfig.resources).toHaveLength(3);
            expect(noteConfig.getResourcesByType(ResourceUsedFor.Community)).toHaveLength(1);
            expect(noteConfig.getResourcesByType(ResourceUsedFor.Social)).toHaveLength(1);
            expect(noteConfig.getResourcesByType(ResourceUsedFor.Feed)).toHaveLength(1);
        });

        it("should handle multilingual resources", () => {
            const config = noteConfigFixtures.variants.multiLanguageNote;

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
            const originalConfig = noteConfigFixtures.variants.externalServiceNote;

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