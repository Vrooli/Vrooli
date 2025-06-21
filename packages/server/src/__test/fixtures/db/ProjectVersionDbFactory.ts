import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import { projectConfigFixtures } from "@vrooli/shared/__test/fixtures/config";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ProjectVersionRelationConfig extends RelationConfig {
    root?: { projectId: string };
    translations?: Array<{ language: string; name: string; description?: string; instructions?: string }>;
    directories?: Array<{
        name: string;
        description?: string;
        parentId?: string;
        childOrder?: number;
    }>;
}

/**
 * Enhanced database fixture factory for ProjectVersion model
 * Handles versioned project content with configurations and directory structures
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Version management (latest/complete flags)
 * - Multi-language translations
 * - Directory/folder structures
 * - Project version settings
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class ProjectVersionDbFactory extends EnhancedDatabaseFactory<
    Prisma.ProjectVersionCreateInput,
    Prisma.ProjectVersionCreateInput,
    Prisma.ProjectVersionInclude,
    Prisma.ProjectVersionUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('ProjectVersion', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.projectVersion;
    }

    /**
     * Get complete test fixtures for ProjectVersion model
     */
    protected getFixtures(): DbTestFixtures<Prisma.ProjectVersionCreateInput, Prisma.ProjectVersionUpdateInput> {
        return {
            minimal: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                isComplete: false,
                isLatest: true,
                isPrivate: false,
                versionLabel: "1.0.0",
                versionIndex: 1,
                projectVersionSettings: projectConfigFixtures.minimal,
            },
            complete: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                isComplete: true,
                isLatest: true,
                isPrivate: false,
                versionLabel: "2.0.0",
                versionIndex: 2,
                versionNotes: "Major update with new features and breaking changes",
                resourceIndex: 1,
                projectVersionSettings: projectConfigFixtures.complete,
                translations: {
                    create: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: "Complete Project Version",
                            description: "A comprehensive project version with all features",
                            instructions: "Follow these detailed instructions to get started",
                        },
                        {
                            id: generatePK().toString(),
                            language: "es",
                            name: "Versión Completa del Proyecto",
                            description: "Una versión completa del proyecto con todas las características",
                            instructions: "Sigue estas instrucciones detalladas para comenzar",
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, publicId, versionLabel, versionIndex
                    isComplete: false,
                    isLatest: true,
                    isPrivate: false,
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    publicId: 123, // Should be string
                    isComplete: "yes", // Should be boolean
                    isLatest: "true", // Should be boolean
                    isPrivate: 1, // Should be boolean
                    versionLabel: 123, // Should be string
                    versionIndex: "one", // Should be number
                    projectVersionSettings: "not an object", // Should be object
                },
                invalidVersionIndex: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    versionIndex: -1, // Should be positive
                    projectVersionSettings: projectConfigFixtures.minimal,
                },
                invalidSettings: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    versionIndex: 1,
                    projectVersionSettings: { invalid: "config" }, // Invalid config structure
                },
            },
            edgeCases: {
                betaVersion: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: false,
                    isLatest: false,
                    isPrivate: true,
                    versionLabel: "3.0.0-beta.1",
                    versionIndex: 3,
                    versionNotes: "Beta release - not for production use",
                    projectVersionSettings: projectConfigFixtures.minimal,
                },
                deprecatedVersion: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: false,
                    isPrivate: false,
                    versionLabel: "0.9.0",
                    versionIndex: 0,
                    versionNotes: "Deprecated - please upgrade to v2.0.0",
                    projectVersionSettings: projectConfigFixtures.minimal,
                },
                openSourceVersion: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    versionIndex: 1,
                    projectVersionSettings: projectConfigFixtures.variants.openSourceProject,
                },
                educationalVersion: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    versionIndex: 1,
                    projectVersionSettings: projectConfigFixtures.variants.educationalProject,
                },
                multiLanguageVersion: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    versionIndex: 1,
                    projectVersionSettings: projectConfigFixtures.variants.multiLanguageProject,
                    translations: {
                        create: Array.from({ length: 5 }, (_, i) => ({
                            id: generatePK().toString(),
                            language: ['en', 'es', 'fr', 'de', 'ja'][i],
                            name: `Project Version ${i}`,
                            description: `Description in language ${i}`,
                        })),
                    },
                },
            },
            updates: {
                minimal: {
                    versionNotes: "Updated version notes",
                },
                complete: {
                    isComplete: true,
                    versionNotes: "Major update with new features",
                    projectVersionSettings: projectConfigFixtures.variants.commercialProject,
                    translations: {
                        update: [{
                            where: { id: "translation_id" },
                            data: { 
                                description: "Updated description",
                                instructions: "Updated instructions",
                            },
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.ProjectVersionCreateInput>): Prisma.ProjectVersionCreateInput {
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            isComplete: false,
            isLatest: true,
            isPrivate: false,
            versionLabel: "1.0.0",
            versionIndex: 1,
            projectVersionSettings: projectConfigFixtures.minimal,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.ProjectVersionCreateInput>): Prisma.ProjectVersionCreateInput {
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            isComplete: true,
            isLatest: true,
            isPrivate: false,
            versionLabel: "2.0.0",
            versionIndex: 2,
            versionNotes: "Major update with new features and breaking changes",
            resourceIndex: 1,
            projectVersionSettings: projectConfigFixtures.complete,
            translations: {
                create: [
                    {
                        id: generatePK().toString(),
                        language: "en",
                        name: "Complete Project Version",
                        description: "A comprehensive project version with all features",
                        instructions: "Follow these detailed instructions to get started",
                    },
                    {
                        id: generatePK().toString(),
                        language: "es",
                        name: "Versión Completa del Proyecto",
                        description: "Una versión completa del proyecto con todas las características",
                        instructions: "Sigue estas instrucciones detalladas para comenzar",
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            latestStableVersion: {
                name: "latestStableVersion",
                description: "Latest stable version of a project",
                config: {
                    overrides: {
                        isLatest: true,
                        isComplete: true,
                        versionLabel: "2.0.0",
                        versionIndex: 2,
                        projectVersionSettings: projectConfigFixtures.complete,
                    },
                    translations: [{
                        language: "en",
                        name: "Latest Stable Version",
                        description: "The most recent stable release",
                    }],
                },
            },
            betaRelease: {
                name: "betaRelease",
                description: "Beta/pre-release version",
                config: {
                    overrides: {
                        isLatest: false,
                        isComplete: false,
                        isPrivate: true,
                        versionLabel: "3.0.0-beta.1",
                        versionIndex: 3,
                        versionNotes: "Beta release - testing new features",
                    },
                    translations: [{
                        language: "en",
                        name: "Beta Release",
                        description: "Pre-release version for testing",
                        instructions: "This is a beta version. Expect bugs and breaking changes.",
                    }],
                },
            },
            openSourceRelease: {
                name: "openSourceRelease",
                description: "Open source project version",
                config: {
                    overrides: {
                        isComplete: true,
                        isLatest: true,
                        projectVersionSettings: projectConfigFixtures.variants.openSourceProject,
                    },
                    translations: [{
                        language: "en",
                        name: "Open Source Release v2.0",
                        description: "Community-driven open source release",
                        instructions: "Check the GitHub repository for contributing guidelines",
                    }],
                    directories: [{
                        name: "src",
                        description: "Source code directory",
                        childOrder: 1,
                    }, {
                        name: "docs",
                        description: "Documentation",
                        childOrder: 2,
                    }, {
                        name: "tests",
                        description: "Test files",
                        childOrder: 3,
                    }],
                },
            },
            educationalRelease: {
                name: "educationalRelease",
                description: "Educational project version",
                config: {
                    overrides: {
                        isComplete: true,
                        isLatest: true,
                        projectVersionSettings: projectConfigFixtures.variants.educationalProject,
                    },
                    translations: [{
                        language: "en",
                        name: "Educational Course v1.0",
                        description: "Interactive learning materials",
                        instructions: "Start with Module 1 and complete all exercises",
                    }],
                    directories: [{
                        name: "Module 1",
                        description: "Introduction to basics",
                        childOrder: 1,
                    }, {
                        name: "Module 2",
                        description: "Intermediate concepts",
                        childOrder: 2,
                    }, {
                        name: "Module 3",
                        description: "Advanced topics",
                        childOrder: 3,
                    }],
                },
            },
            versionWithDirectories: {
                name: "versionWithDirectories",
                description: "Version with complex directory structure",
                config: {
                    overrides: {
                        isComplete: true,
                        isLatest: true,
                    },
                    translations: [{
                        language: "en",
                        name: "Project with Directories",
                        description: "Complex project structure",
                    }],
                    directories: [{
                        name: "root",
                        description: "Root directory",
                        childOrder: 1,
                    }],
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.ProjectVersionInclude {
        return {
            root: {
                select: {
                    id: true,
                    publicId: true,
                    handle: true,
                    isPrivate: true,
                },
            },
            translations: true,
            directories: {
                select: {
                    id: true,
                    publicId: true,
                    childOrder: true,
                    isRoot: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                            description: true,
                        },
                    },
                },
                orderBy: {
                    childOrder: 'asc',
                },
            },
            _count: {
                select: {
                    translations: true,
                    directories: true,
                    routineVersions: true,
                    dataStructureVersions: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.ProjectVersionCreateInput,
        config: ProjectVersionRelationConfig,
        tx: any
    ): Promise<Prisma.ProjectVersionCreateInput> {
        let data = { ...baseData };

        // Handle root project connection
        if (config.root?.projectId) {
            data.root = { connect: { id: config.root.projectId } };
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK().toString(),
                    ...trans,
                })),
            };
        }

        // Handle directories (project structure)
        if (config.directories && Array.isArray(config.directories)) {
            data.directories = {
                create: config.directories.map(dir => ({
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isRoot: dir.parentId === undefined,
                    childOrder: dir.childOrder ?? 0,
                    parentDirectoryId: dir.parentId,
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            name: dir.name,
                            description: dir.description,
                        }],
                    },
                })),
            };
        }

        return data;
    }

    /**
     * Create an open source version
     */
    async createOpenSourceVersion(projectId: string): Promise<Prisma.ProjectVersion> {
        return this.createWithRelations({
            root: { projectId },
            overrides: {
                isComplete: true,
                isLatest: true,
                projectVersionSettings: projectConfigFixtures.variants.openSourceProject,
            },
            translations: [{
                language: "en",
                name: "Open Source Release v2.0",
                description: "Community-driven open source release",
                instructions: "Check the GitHub repository for contributing guidelines",
            }],
        });
    }

    /**
     * Create an educational version
     */
    async createEducationalVersion(projectId: string): Promise<Prisma.ProjectVersion> {
        return this.seedScenario('educationalRelease');
    }

    /**
     * Create a beta/pre-release version
     */
    async createBetaVersion(projectId: string): Promise<Prisma.ProjectVersion> {
        return this.createWithRelations({
            root: { projectId },
            overrides: {
                isLatest: false,
                isComplete: false,
                isPrivate: true,
                versionLabel: "3.0.0-beta.1",
                versionIndex: 3,
                versionNotes: "Beta release - testing new features",
            },
            translations: [{
                language: "en",
                name: "Beta Release",
                description: "Pre-release version for testing",
                instructions: "This is a beta version. Expect bugs and breaking changes.",
            }],
        });
    }

    /**
     * Create version with directory structure
     */
    async createVersionWithDirectories(projectId: string): Promise<Prisma.ProjectVersion> {
        return this.createWithRelations({
            root: { projectId },
            overrides: {
                isComplete: true,
                isLatest: true,
            },
            translations: [{
                language: "en",
                name: "Structured Project",
                description: "Project with organized directory structure",
            }],
            directories: [
                { name: "src", description: "Source code", childOrder: 1 },
                { name: "docs", description: "Documentation", childOrder: 2 },
                { name: "tests", description: "Test files", childOrder: 3 },
                { name: "examples", description: "Example code", childOrder: 4 },
            ],
        });
    }

    protected async checkModelConstraints(record: Prisma.ProjectVersion): Promise<string[]> {
        const violations: string[] = [];

        // Check version index is positive
        if (record.versionIndex < 0) {
            violations.push('Version index must be non-negative');
        }

        // Check that version belongs to a project
        if (!record.rootId) {
            violations.push('ProjectVersion must belong to a Project');
        }

        // Check version label format
        if (record.versionLabel && !/^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$/.test(record.versionLabel)) {
            violations.push('Version label should follow semantic versioning format');
        }

        // Check that only one version is marked as latest per project
        if (record.isLatest && record.rootId) {
            const otherLatest = await this.prisma.projectVersion.count({
                where: {
                    rootId: record.rootId,
                    isLatest: true,
                    id: { not: record.id },
                },
            });
            if (otherLatest > 0) {
                violations.push('Only one version can be marked as latest per project');
            }
        }

        // Check projectVersionSettings structure
        if (record.projectVersionSettings && 
            (!record.projectVersionSettings.__version || 
             typeof record.projectVersionSettings.__version !== 'string')) {
            violations.push('Project version settings must have a valid __version field');
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            directories: {
                include: {
                    translations: true,
                    childDirectories: true,
                },
            },
            routineVersions: true,
            dataStructureVersions: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.ProjectVersion,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete translations
        if (shouldDelete('translations') && record.translations?.length) {
            await tx.projectVersionTranslation.deleteMany({
                where: { projectVersionId: record.id },
            });
        }

        // Delete directories and their translations
        if (shouldDelete('directories') && record.directories?.length) {
            for (const dir of record.directories) {
                if (dir.translations?.length) {
                    await tx.projectVersionDirectoryTranslation.deleteMany({
                        where: { projectVersionDirectoryId: dir.id },
                    });
                }
            }
            
            await tx.projectVersionDirectory.deleteMany({
                where: { projectVersionId: record.id },
            });
        }

        // Remove associations with routine versions
        if (shouldDelete('routineVersions') && record.routineVersions?.length) {
            await tx.routineVersion.updateMany({
                where: { projectVersionId: record.id },
                data: { projectVersionId: null },
            });
        }

        // Remove associations with data structure versions
        if (shouldDelete('dataStructureVersions') && record.dataStructureVersions?.length) {
            await tx.dataStructureVersion.updateMany({
                where: { projectVersionId: record.id },
                data: { projectVersionId: null },
            });
        }
    }

    /**
     * Create version history for testing
     */
    async createVersionHistory(projectId: string, count: number = 3): Promise<Prisma.ProjectVersion[]> {
        const versions: Prisma.ProjectVersion[] = [];
        const configs = [
            projectConfigFixtures.minimal,
            projectConfigFixtures.variants.openSourceProject,
            projectConfigFixtures.complete,
        ];
        
        for (let i = 0; i < count; i++) {
            const isLatest = i === count - 1;
            const version = await this.createWithRelations({
                root: { projectId },
                overrides: {
                    versionLabel: `${i + 1}.0.0`,
                    versionIndex: i + 1,
                    isLatest,
                    isComplete: true,
                    versionNotes: isLatest ? "Latest version" : `Version ${i + 1} release notes`,
                    projectVersionSettings: configs[i % configs.length],
                },
                translations: [{
                    language: "en",
                    name: `Version ${i + 1}.0.0`,
                    description: isLatest ? "Latest version with all features" : `Version ${i + 1} of the project`,
                }],
            });
            versions.push(version);
        }
        
        return versions;
    }
}

// Export factory creator function
export const createProjectVersionDbFactory = (prisma: PrismaClient) => 
    ProjectVersionDbFactory.getInstance('ProjectVersion', prisma);

// Export the class for type usage
export { ProjectVersionDbFactory as ProjectVersionDbFactoryClass };