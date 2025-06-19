import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for ResourceVersion model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const resourceVersionDbIds = {
    version1: generatePK(),
    version2: generatePK(),
    version3: generatePK(),
    version4: generatePK(),
    version5: generatePK(),
    resource1: generatePK(),
    resource2: generatePK(),
    resource3: generatePK(),
    translation1: generatePK(),
    translation2: generatePK(),
    translation3: generatePK(),
    translation4: generatePK(),
    relation1: generatePK(),
    relation2: generatePK(),
};

/**
 * Minimal resource version data for database creation
 */
export const minimalResourceVersionDb: Prisma.resource_versionCreateInput = {
    id: resourceVersionDbIds.version1,
    publicId: generatePublicId(),
    versionLabel: "1.0.0",
    isComplete: false,
    isDeleted: false,
    isLatest: true,
    isLatestPublic: true,
    isPrivate: false,
    root: { connect: { id: resourceVersionDbIds.resource1 } },
    translations: {
        create: [{
            id: resourceVersionDbIds.translation1,
            language: "en",
            name: "Test Resource Version",
        }],
    },
};

/**
 * Complete resource version with all features
 */
export const completeResourceVersionDb: Prisma.resource_versionCreateInput = {
    id: resourceVersionDbIds.version2,
    publicId: generatePublicId(),
    versionLabel: "2.1.0",
    versionNotes: "Major update with new features and improvements",
    isComplete: true,
    isDeleted: false,
    isLatest: true,
    isLatestPublic: true,
    isPrivate: false,
    intendToPullRequest: false,
    codeLanguage: "typescript",
    resourceSubType: "RoutineGenerate",
    isAutomatable: true,
    complexity: 5,
    config: {
        version: "2.1.0",
        features: ["automation", "validation"],
        settings: {
            timeout: 30000,
            retries: 3,
        },
    },
    root: { connect: { id: resourceVersionDbIds.resource2 } },
    translations: {
        create: [
            {
                id: resourceVersionDbIds.translation2,
                language: "en",
                name: "Complete Resource Version",
                description: "A comprehensive resource version with all features enabled",
                details: "This version includes advanced configuration options and automation capabilities",
                instructions: "Follow the setup guide carefully and ensure all dependencies are installed",
            },
            {
                id: resourceVersionDbIds.translation3,
                language: "es",
                name: "Versión Completa del Recurso",
                description: "Una versión integral del recurso con todas las características habilitadas",
            },
        ],
    },
    relatedVersionsFrom: {
        create: [{
            id: resourceVersionDbIds.relation1,
            toVersion: { connect: { id: resourceVersionDbIds.version3 } },
            relationType: "DEPENDENCY",
            labels: ["required", "api"],
        }],
    },
};

/**
 * Private resource version
 */
export const privateResourceVersionDb: Prisma.resource_versionCreateInput = {
    id: resourceVersionDbIds.version3,
    publicId: generatePublicId(),
    versionLabel: "1.0.0-private",
    isComplete: true,
    isDeleted: false,
    isLatest: true,
    isLatestPublic: false,
    isPrivate: true,
    root: { connect: { id: resourceVersionDbIds.resource3 } },
    translations: {
        create: [{
            id: resourceVersionDbIds.translation4,
            language: "en",
            name: "Private Resource Version",
            description: "A private resource version for internal use only",
        }],
    },
};

/**
 * Code resource version
 */
export const codeResourceVersionDb: Prisma.resource_versionCreateInput = {
    id: resourceVersionDbIds.version4,
    publicId: generatePublicId(),
    versionLabel: "1.2.0",
    codeLanguage: "python",
    resourceSubType: "CodeDataConverter",
    isComplete: true,
    isDeleted: false,
    isLatest: true,
    isLatestPublic: true,
    isPrivate: false,
    complexity: 3,
    config: {
        language: "python",
        runtime: "3.9",
        dependencies: ["pandas", "numpy"],
    },
    root: { connect: { id: resourceVersionDbIds.resource1 } },
    translations: {
        create: [{
            id: generatePK(),
            language: "en",
            name: "Python Data Converter",
            description: "A Python script for data format conversion",
            instructions: "Install required dependencies before running",
        }],
    },
};

/**
 * Factory for creating resource version database fixtures with overrides
 */
export class ResourceVersionDbFactory {
    static createMinimal(overrides?: Partial<Prisma.resource_versionCreateInput>): Prisma.resource_versionCreateInput {
        return {
            ...minimalResourceVersionDb,
            id: generatePK(),
            publicId: generatePublicId(),
            versionLabel: `1.0.${Math.floor(Math.random() * 100)}`,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: `Test Resource Version ${nanoid(4)}`,
                }],
            },
            ...overrides,
        };
    }

    static createComplete(overrides?: Partial<Prisma.resource_versionCreateInput>): Prisma.resource_versionCreateInput {
        return {
            ...completeResourceVersionDb,
            id: generatePK(),
            publicId: generatePublicId(),
            versionLabel: `2.${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}`,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: `Complete Resource Version ${nanoid(4)}`,
                    description: "A comprehensive resource version with all features enabled",
                }],
            },
            ...overrides,
        };
    }

    static createPrivate(overrides?: Partial<Prisma.resource_versionCreateInput>): Prisma.resource_versionCreateInput {
        return {
            ...privateResourceVersionDb,
            id: generatePK(),
            publicId: generatePublicId(),
            versionLabel: `1.0.${Math.floor(Math.random() * 100)}-private`,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: `Private Resource Version ${nanoid(4)}`,
                }],
            },
            ...overrides,
        };
    }

    static createCode(
        language: string = "javascript",
        overrides?: Partial<Prisma.resource_versionCreateInput>
    ): Prisma.resource_versionCreateInput {
        return {
            ...codeResourceVersionDb,
            id: generatePK(),
            publicId: generatePublicId(),
            codeLanguage: language,
            versionLabel: `1.${Math.floor(Math.random() * 10)}.0`,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: `${language.charAt(0).toUpperCase() + language.slice(1)} Code Version ${nanoid(4)}`,
                    description: `A ${language} code resource`,
                }],
            },
            ...overrides,
        };
    }

    static createWithRelations(
        relations: Array<{ toVersionId: string; relationType: string; labels: string[] }>,
        overrides?: Partial<Prisma.resource_versionCreateInput>
    ): Prisma.resource_versionCreateInput {
        return {
            ...this.createComplete(overrides),
            relatedVersionsFrom: {
                create: relations.map(rel => ({
                    id: generatePK(),
                    toVersion: { connect: { id: rel.toVersionId } },
                    relationType: rel.relationType,
                    labels: rel.labels,
                })),
            },
        };
    }

    static createWithTranslations(
        translations: Array<{ language: string; name: string; description?: string }>,
        overrides?: Partial<Prisma.resource_versionCreateInput>
    ): Prisma.resource_versionCreateInput {
        return {
            ...this.createMinimal(overrides),
            translations: {
                create: translations.map(t => ({
                    id: generatePK(),
                    language: t.language,
                    name: t.name,
                    description: t.description,
                })),
            },
        };
    }

    /**
     * Create routine resource version
     */
    static createRoutine(
        isAutomatable: boolean = true,
        overrides?: Partial<Prisma.resource_versionCreateInput>
    ): Prisma.resource_versionCreateInput {
        return {
            ...this.createComplete(overrides),
            resourceSubType: "RoutineGenerate",
            isAutomatable,
            complexity: Math.floor(Math.random() * 10) + 1,
            config: {
                type: "routine",
                automation: { enabled: isAutomatable },
                steps: ["input", "process", "output"],
            },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: `${isAutomatable ? "Automated" : "Manual"} Routine ${nanoid(4)}`,
                    description: `A ${isAutomatable ? "fully automated" : "manual"} routine for processing tasks`,
                    instructions: isAutomatable ? "This routine runs automatically" : "Manual execution required",
                }],
            },
        };
    }

    /**
     * Create API resource version
     */
    static createApi(overrides?: Partial<Prisma.resource_versionCreateInput>): Prisma.resource_versionCreateInput {
        return {
            ...this.createComplete(overrides),
            resourceSubType: "ApiGenerate",
            config: {
                type: "api",
                version: "v1",
                endpoints: ["/data", "/status"],
                authentication: "bearer",
            },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: `API Resource ${nanoid(4)}`,
                    description: "A REST API resource with multiple endpoints",
                    instructions: "Authenticate using Bearer token before making requests",
                }],
            },
        };
    }
}

/**
 * Helper to seed resource versions for testing
 */
export async function seedResourceVersions(
    prisma: any,
    options: {
        rootId: string;
        count?: number;
        withTranslations?: boolean;
        withRelations?: boolean;
        languages?: string[];
        resourceType?: "code" | "routine" | "api" | "standard";
    }
) {
    const { rootId, count = 3, withTranslations = true, languages = ["en"] } = options;
    const versions = [];

    for (let i = 0; i < count; i++) {
        let versionData: Prisma.resource_versionCreateInput;

        switch (options.resourceType) {
            case "code":
                versionData = ResourceVersionDbFactory.createCode("typescript", {
                    root: { connect: { id: rootId } },
                });
                break;
            case "routine":
                versionData = ResourceVersionDbFactory.createRoutine(true, {
                    root: { connect: { id: rootId } },
                });
                break;
            case "api":
                versionData = ResourceVersionDbFactory.createApi({
                    root: { connect: { id: rootId } },
                });
                break;
            default:
                versionData = ResourceVersionDbFactory.createComplete({
                    root: { connect: { id: rootId } },
                });
        }

        // Update version label to be unique
        versionData.versionLabel = `${i + 1}.0.0`;
        versionData.isLatest = i === count - 1; // Only last version is latest
        versionData.isLatestPublic = i === count - 1 && !versionData.isPrivate;

        // Add multiple language translations if requested
        if (withTranslations && languages.length > 1) {
            versionData.translations = {
                create: languages.map(lang => ({
                    id: generatePK(),
                    language: lang,
                    name: `${versionData.translations?.create?.[0]?.name || "Resource Version"} (${lang})`,
                    description: lang === "en" ? "A test resource version" : 
                               lang === "es" ? "Una versión de recurso de prueba" :
                               lang === "fr" ? "Une version de ressource de test" :
                               "A test resource version",
                })),
            };
        }

        const version = await prisma.resource_version.create({
            data: versionData,
            include: {
                translations: true,
                relatedVersionsFrom: true,
                relatedVersionsTo: true,
            },
        });

        versions.push(version);
    }

    // Add relations between versions if requested
    if (options.withRelations && versions.length > 1) {
        await prisma.resource_version_relation.create({
            data: {
                id: generatePK(),
                fromVersion: { connect: { id: versions[0].id } },
                toVersion: { connect: { id: versions[1].id } },
                relationType: "SUCCESSOR",
                labels: ["upgrade", "improved"],
            },
        });
    }

    return versions;
}

/**
 * Helper to clean up resource versions and related data
 */
export async function cleanupResourceVersions(prisma: any, versionIds: string[]) {
    // Clean up in correct order due to foreign key constraints
    await prisma.resource_version_relation.deleteMany({
        where: {
            OR: [
                { fromVersionId: { in: versionIds } },
                { toVersionId: { in: versionIds } },
            ],
        },
    });

    await prisma.resource_translation.deleteMany({
        where: { resourceVersionId: { in: versionIds } },
    });

    await prisma.resource_version.deleteMany({
        where: { id: { in: versionIds } },
    });
}