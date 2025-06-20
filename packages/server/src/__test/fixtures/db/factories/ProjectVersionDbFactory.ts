import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";
import { projectConfigFixtures } from "@vrooli/shared/__test/fixtures/config";

interface ProjectVersionRelationConfig extends RelationConfig {
    project?: { projectId: string };
    parent?: { parentId: string };
    directories?: Array<{
        name: string;
        isRoot?: boolean;
        childOrder?: number;
        translations?: Array<{ language: string; name: string; description?: string }>;
    }>;
    resourceLists?: Array<{
        name: string;
        description?: string;
        isPrivate?: boolean;
    }>;
    translations?: Array<{ language: string; name: string; description?: string; instructions?: string; details?: string }>;
}

/**
 * Database fixture factory for ProjectVersion model
 * Handles versioned project content with directories and resource lists
 */
export class ProjectVersionDbFactory extends DatabaseFixtureFactory<
    Prisma.ProjectVersion,
    Prisma.ProjectVersionCreateInput,
    Prisma.ProjectVersionInclude,
    Prisma.ProjectVersionUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('ProjectVersion', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.projectVersion;
    }

    protected getMinimalData(overrides?: Partial<Prisma.ProjectVersionCreateInput>): Prisma.ProjectVersionCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            versionLabel: "1.0.0",
            versionIndex: 1,
            isLatest: true,
            isComplete: true,
            isPrivate: false,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.ProjectVersionCreateInput>): Prisma.ProjectVersionCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            versionLabel: "1.0.0",
            versionIndex: 1,
            isLatest: true,
            isComplete: true,
            isPrivate: false,
            config: projectConfigFixtures.complete,
            complexity: 5,
            translations: {
                create: [
                    {
                        id: generatePK(),
                        language: "en",
                        name: "Complete Project Version",
                        description: "A comprehensive project version with all features",
                        instructions: "Follow these detailed instructions to use this project",
                        details: "Detailed information about this project version",
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        name: "Versión Completa del Proyecto",
                        description: "Una versión de proyecto integral con todas las funcionalidades",
                        instructions: "Sigue estas instrucciones detalladas para usar este proyecto",
                        details: "Información detallada sobre esta versión del proyecto",
                    },
                ],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.ProjectVersionInclude {
        return {
            translations: true,
            project: {
                select: {
                    id: true,
                    publicId: true,
                    handle: true,
                    isPrivate: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                        },
                    },
                },
            },
            parent: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                },
            },
            children: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    versionIndex: true,
                },
                orderBy: {
                    versionIndex: 'asc',
                },
            },
            directories: {
                select: {
                    id: true,
                    publicId: true,
                    isRoot: true,
                    childOrder: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                        },
                    },
                },
                orderBy: {
                    childOrder: 'asc',
                },
            },
            resourceLists: {
                select: {
                    id: true,
                    publicId: true,
                    isPrivate: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                            description: true,
                        },
                    },
                },
            },
            _count: {
                select: {
                    children: true,
                    directories: true,
                    resourceLists: true,
                    bookmarks: true,
                    views: true,
                    votes: true,
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

        // Handle project connection
        if (config.project) {
            data.project = {
                connect: { id: config.project.projectId },
            };
        }

        // Handle parent version
        if (config.parent) {
            data.parent = {
                connect: { id: config.parent.parentId },
            };
        }

        // Handle directories
        if (config.directories && Array.isArray(config.directories)) {
            data.directories = {
                create: config.directories.map((directory, index) => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isRoot: directory.isRoot ?? (index === 0),
                    childOrder: directory.childOrder ?? index,
                    translations: directory.translations ? {
                        create: directory.translations.map(trans => ({
                            id: generatePK(),
                            ...trans,
                        })),
                    } : {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: directory.name,
                            description: directory.name,
                        }],
                    },
                })),
            };
        }

        // Handle resource lists
        if (config.resourceLists && Array.isArray(config.resourceLists)) {
            data.resourceLists = {
                create: config.resourceLists.map(resourceList => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: resourceList.isPrivate ?? false,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: resourceList.name,
                            description: resourceList.description ?? resourceList.name,
                        }],
                    },
                })),
            };
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK(),
                    ...trans,
                })),
            };
        }

        return data;
    }

    /**
     * Create a draft version
     */
    async createDraftVersion(): Promise<Prisma.ProjectVersion> {
        return this.createWithRelations({
            overrides: {
                versionLabel: "0.1.0-draft",
                isComplete: false,
                isPrivate: true,
                isLatest: false,
            },
            translations: [
                {
                    language: "en",
                    name: "Draft Project Version",
                    description: "Work in progress project version",
                    instructions: "This is a draft version under development",
                },
            ],
        });
    }

    /**
     * Create a published version with directories
     */
    async createPublishedVersionWithDirectories(): Promise<Prisma.ProjectVersion> {
        return this.createWithRelations({
            overrides: {
                versionLabel: "1.0.0",
                isComplete: true,
                isPrivate: false,
                isLatest: true,
                complexity: 7,
            },
            directories: [
                {
                    name: "Root Directory",
                    isRoot: true,
                    childOrder: 0,
                    translations: [
                        {
                            language: "en",
                            name: "Root Directory",
                            description: "Main project directory",
                        },
                    ],
                },
                {
                    name: "Documentation",
                    isRoot: false,
                    childOrder: 1,
                    translations: [
                        {
                            language: "en",
                            name: "Documentation",
                            description: "Project documentation and guides",
                        },
                    ],
                },
                {
                    name: "Resources",
                    isRoot: false,
                    childOrder: 2,
                    translations: [
                        {
                            language: "en",
                            name: "Resources",
                            description: "Additional project resources",
                        },
                    ],
                },
            ],
            resourceLists: [
                {
                    name: "Getting Started",
                    description: "Resources for getting started with this project",
                    isPrivate: false,
                },
                {
                    name: "Advanced Topics",
                    description: "Advanced project usage and customization",
                    isPrivate: false,
                },
            ],
            translations: [
                {
                    language: "en",
                    name: "Published Project Version",
                    description: "A complete published version",
                    instructions: "Complete instructions for using this project",
                    details: "Detailed information about features and capabilities",
                },
            ],
        });
    }

    /**
     * Create a deprecated version
     */
    async createDeprecatedVersion(): Promise<Prisma.ProjectVersion> {
        return this.createWithRelations({
            overrides: {
                versionLabel: "0.9.0",
                isComplete: true,
                isPrivate: false,
                isLatest: false,
                versionIndex: 1,
            },
            translations: [
                {
                    language: "en",
                    name: "Deprecated Project Version",
                    description: "This version is deprecated, use a newer version",
                    instructions: "Please upgrade to the latest version",
                },
            ],
        });
    }

    /**
     * Create a child version (fork/branch)
     */
    async createChildVersion(parentId: string): Promise<Prisma.ProjectVersion> {
        return this.createWithRelations({
            parent: { parentId },
            overrides: {
                versionLabel: "1.1.0-fork",
                isComplete: false,
                isPrivate: true,
                isLatest: false,
                versionIndex: 1,
            },
            translations: [
                {
                    language: "en",
                    name: "Forked Project Version",
                    description: "A fork of the parent project",
                    instructions: "Based on parent version with modifications",
                },
            ],
        });
    }

    protected async checkModelConstraints(record: Prisma.ProjectVersion): Promise<string[]> {
        const violations: string[] = [];
        
        // Check that projectId is valid
        if (record.projectId) {
            const project = await this.prisma.project.findUnique({
                where: { id: record.projectId },
            });
            if (!project) {
                violations.push('ProjectVersion must belong to a valid project');
            }
        }

        // Check version label format
        if (record.versionLabel && !/^\d+\.\d+\.\d+(-[\w.-]+)?$/.test(record.versionLabel)) {
            violations.push('Version label must follow semantic versioning format');
        }

        // Check that only one version per project is marked as latest
        if (record.isLatest && record.projectId) {
            const otherLatestVersions = await this.prisma.projectVersion.count({
                where: {
                    projectId: record.projectId,
                    isLatest: true,
                    id: { not: record.id },
                },
            });
            
            if (otherLatestVersions > 0) {
                violations.push('Only one version per project can be marked as latest');
            }
        }

        // Check that completed versions are not private if project is public
        if (record.isComplete && !record.isPrivate && record.projectId) {
            const project = await this.prisma.project.findUnique({
                where: { id: record.projectId },
                select: { isPrivate: true },
            });
            
            if (project && !project.isPrivate && record.isPrivate) {
                violations.push('Complete versions of public projects should not be private');
            }
        }

        // Check complexity range
        if (record.complexity !== null && record.complexity !== undefined) {
            if (record.complexity < 1 || record.complexity > 10) {
                violations.push('Complexity must be between 1 and 10');
            }
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, publicId, versionLabel, versionIndex
                isLatest: true,
                isComplete: true,
                isPrivate: false,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                versionLabel: true, // Should be string
                versionIndex: "1", // Should be number
                isLatest: "yes", // Should be boolean
                isComplete: 1, // Should be boolean
                isPrivate: "no", // Should be boolean
            },
            invalidVersionFormat: {
                id: generatePK(),
                publicId: generatePublicId(),
                versionLabel: "invalid-version", // Invalid format
                versionIndex: 1,
                isLatest: true,
                isComplete: true,
                isPrivate: false,
            },
            invalidComplexity: {
                id: generatePK(),
                publicId: generatePublicId(),
                versionLabel: "1.0.0",
                versionIndex: 1,
                isLatest: true,
                isComplete: true,
                isPrivate: false,
                complexity: 15, // Out of range
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.ProjectVersionCreateInput> {
        return {
            prereleaseVersion: {
                ...this.getMinimalData(),
                versionLabel: "2.0.0-alpha.1",
                isComplete: false,
                isPrivate: true,
            },
            buildMetadataVersion: {
                ...this.getMinimalData(),
                versionLabel: "1.0.0+build.123",
                isComplete: true,
                isPrivate: false,
            },
            maxComplexity: {
                ...this.getMinimalData(),
                versionLabel: "1.0.0",
                complexity: 10,
                config: {
                    ...projectConfigFixtures.complete,
                    complexity: 10,
                },
            },
            minimalVersion: {
                ...this.getMinimalData(),
                versionLabel: "0.0.1",
                complexity: 1,
                config: projectConfigFixtures.minimal,
            },
            unicodeContent: {
                ...this.getMinimalData(),
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "zh",
                        name: "项目版本 🚀",
                        description: "使用Unicode字符的项目版本",
                        instructions: "详细说明如何使用这个项目",
                        details: "项目的详细信息和功能描述",
                    }],
                },
            },
            largeDirectoryStructure: {
                ...this.getMinimalData(),
                // Would need to create many directories after creation
                // due to complexity of nested create operations
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            directories: {
                include: {
                    translations: true,
                    children: true,
                },
            },
            resourceLists: {
                include: {
                    translations: true,
                    resources: true,
                },
            },
            children: true,
            bookmarks: true,
            views: true,
            votes: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.ProjectVersion,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Delete child versions first
        if (record.children?.length) {
            await tx.projectVersion.deleteMany({
                where: { parentId: record.id },
            });
        }

        // Delete directories (cascade will handle their translations)
        if (record.directories?.length) {
            await tx.projectVersionDirectory.deleteMany({
                where: { projectVersionId: record.id },
            });
        }

        // Delete resource lists (cascade will handle their resources)
        if (record.resourceLists?.length) {
            await tx.resourceList.deleteMany({
                where: { projectVersionId: record.id },
            });
        }

        // Delete bookmarks
        if (record.bookmarks?.length) {
            await tx.bookmark.deleteMany({
                where: { projectVersionId: record.id },
            });
        }

        // Delete views
        if (record.views?.length) {
            await tx.view.deleteMany({
                where: { projectVersionId: record.id },
            });
        }

        // Delete votes/reactions
        if (record.votes?.length) {
            await tx.reaction.deleteMany({
                where: { projectVersionId: record.id },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.projectVersionTranslation.deleteMany({
                where: { projectVersionId: record.id },
            });
        }
    }

    /**
     * Create version sequence for a project
     */
    async createVersionSequence(
        projectId: string,
        versions: Array<{ label: string; isLatest?: boolean; isComplete?: boolean }>
    ): Promise<Prisma.ProjectVersion[]> {
        const createdVersions: Prisma.ProjectVersion[] = [];
        
        for (let i = 0; i < versions.length; i++) {
            const version = versions[i];
            const projectVersion = await this.createWithRelations({
                project: { projectId },
                overrides: {
                    versionLabel: version.label,
                    versionIndex: i + 1,
                    isLatest: version.isLatest ?? (i === versions.length - 1),
                    isComplete: version.isComplete ?? true,
                },
                translations: [
                    {
                        language: "en",
                        name: `Project Version ${version.label}`,
                        description: `Version ${version.label} of the project`,
                    },
                ],
            });
            createdVersions.push(projectVersion);
        }
        
        return createdVersions;
    }
}

// Export factory creator function
export const createProjectVersionDbFactory = (prisma: PrismaClient) => new ProjectVersionDbFactory(prisma);