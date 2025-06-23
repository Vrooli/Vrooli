import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for ResourceVersionRelation model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent test IDs for resource version relations
export const resourceVersionRelationDbIds = {
    relation1: generatePK(),
    relation2: generatePK(),
    relation3: generatePK(),
    relation4: generatePK(),
    relation5: generatePK(),
    fromVersion1: generatePK(),
    fromVersion2: generatePK(),
    toVersion1: generatePK(),
    toVersion2: generatePK(),
    toVersion3: generatePK(),
};

/**
 * Minimal resource version relation data for database creation
 */
export const minimalResourceVersionRelationDb: Prisma.resource_version_relationCreateInput = {
    id: resourceVersionRelationDbIds.relation1,
    relationType: "DEPENDENCY",
    labels: [],
    fromVersion: { connect: { id: resourceVersionRelationDbIds.fromVersion1 } },
    toVersion: { connect: { id: resourceVersionRelationDbIds.toVersion1 } },
};

/**
 * Complete resource version relation with all features
 */
export const completeResourceVersionRelationDb: Prisma.resource_version_relationCreateInput = {
    id: resourceVersionRelationDbIds.relation2,
    relationType: "SUBROUTINE",
    labels: ["dependency", "upgrade", "replaces"],
    fromVersion: { connect: { id: resourceVersionRelationDbIds.fromVersion1 } },
    toVersion: { connect: { id: resourceVersionRelationDbIds.toVersion2 } },
};

/**
 * API call relation example
 */
export const apiCallRelationDb: Prisma.resource_version_relationCreateInput = {
    id: resourceVersionRelationDbIds.relation3,
    relationType: "API_CALL",
    labels: ["api", "external", "required"],
    fromVersion: { connect: { id: resourceVersionRelationDbIds.fromVersion1 } },
    toVersion: { connect: { id: resourceVersionRelationDbIds.toVersion3 } },
};

/**
 * Code call relation example
 */
export const codeCallRelationDb: Prisma.resource_version_relationCreateInput = {
    id: resourceVersionRelationDbIds.relation4,
    relationType: "CODE_CALL",
    labels: ["library", "import", "function"],
    fromVersion: { connect: { id: resourceVersionRelationDbIds.fromVersion2 } },
    toVersion: { connect: { id: resourceVersionRelationDbIds.toVersion1 } },
};

/**
 * Factory for creating resource version relation database fixtures with overrides
 */
export class ResourceVersionRelationDbFactory {
    /**
     * Create minimal resource version relation
     */
    static createMinimal(
        fromVersionId: string,
        toVersionId: string,
        overrides?: Partial<Prisma.resource_version_relationCreateInput>,
    ): Prisma.resource_version_relationCreateInput {
        return {
            id: generatePK(),
            relationType: "DEPENDENCY",
            labels: [],
            fromVersion: { connect: { id: fromVersionId } },
            toVersion: { connect: { id: toVersionId } },
            ...overrides,
        };
    }

    /**
     * Create complete resource version relation with labels
     */
    static createComplete(
        fromVersionId: string,
        toVersionId: string,
        relationType = "SUBROUTINE",
        labels: string[] = ["dependency", "upgrade"],
        overrides?: Partial<Prisma.resource_version_relationCreateInput>,
    ): Prisma.resource_version_relationCreateInput {
        return {
            id: generatePK(),
            relationType,
            labels,
            fromVersion: { connect: { id: fromVersionId } },
            toVersion: { connect: { id: toVersionId } },
            ...overrides,
        };
    }

    /**
     * Create API call relation
     */
    static createApiCall(
        fromVersionId: string,
        toVersionId: string,
        labels: string[] = ["api", "external"],
        overrides?: Partial<Prisma.resource_version_relationCreateInput>,
    ): Prisma.resource_version_relationCreateInput {
        return this.createComplete(fromVersionId, toVersionId, "API_CALL", labels, overrides);
    }

    /**
     * Create code call relation
     */
    static createCodeCall(
        fromVersionId: string,
        toVersionId: string,
        labels: string[] = ["library", "import"],
        overrides?: Partial<Prisma.resource_version_relationCreateInput>,
    ): Prisma.resource_version_relationCreateInput {
        return this.createComplete(fromVersionId, toVersionId, "CODE_CALL", labels, overrides);
    }

    /**
     * Create subroutine relation
     */
    static createSubroutine(
        fromVersionId: string,
        toVersionId: string,
        labels: string[] = ["subroutine", "dependency"],
        overrides?: Partial<Prisma.resource_version_relationCreateInput>,
    ): Prisma.resource_version_relationCreateInput {
        return this.createComplete(fromVersionId, toVersionId, "SUBROUTINE", labels, overrides);
    }

    /**
     * Create upgrade relation (version evolution)
     */
    static createUpgrade(
        fromVersionId: string,
        toVersionId: string,
        labels: string[] = ["upgrade", "successor"],
        overrides?: Partial<Prisma.resource_version_relationCreateInput>,
    ): Prisma.resource_version_relationCreateInput {
        return this.createComplete(fromVersionId, toVersionId, "UPGRADE", labels, overrides);
    }

    /**
     * Create replacement relation
     */
    static createReplacement(
        fromVersionId: string,
        toVersionId: string,
        labels: string[] = ["replaces", "deprecated"],
        overrides?: Partial<Prisma.resource_version_relationCreateInput>,
    ): Prisma.resource_version_relationCreateInput {
        return this.createComplete(fromVersionId, toVersionId, "REPLACES", labels, overrides);
    }
}

/**
 * Helper to seed multiple resource version relations for testing
 */
export async function seedResourceVersionRelations(
    prisma: any,
    options: {
        fromVersionIds: string[];
        toVersionIds: string[];
        relationType?: string;
        labels?: string[];
        count?: number;
    },
) {
    const relations = [];
    const count = options.count || Math.min(options.fromVersionIds.length, options.toVersionIds.length);

    for (let i = 0; i < count; i++) {
        const fromVersionId = options.fromVersionIds[i % options.fromVersionIds.length];
        const toVersionId = options.toVersionIds[i % options.toVersionIds.length];
        
        const relationData = ResourceVersionRelationDbFactory.createComplete(
            fromVersionId,
            toVersionId,
            options.relationType,
            options.labels,
        );

        const relation = await prisma.resource_version_relation.create({
            data: relationData,
            include: {
                fromVersion: true,
                toVersion: true,
            },
        });
        
        relations.push(relation);
    }

    return relations;
}

/**
 * Helper to create a chain of related resource versions (A -> B -> C)
 */
export async function createResourceVersionRelationChain(
    prisma: any,
    versionIds: string[],
    relationType = "DEPENDENCY",
    labels: string[] = ["dependency"],
) {
    const relations = [];

    for (let i = 0; i < versionIds.length - 1; i++) {
        const relationData = ResourceVersionRelationDbFactory.createComplete(
            versionIds[i],
            versionIds[i + 1],
            relationType,
            labels,
        );

        const relation = await prisma.resource_version_relation.create({
            data: relationData,
            include: {
                fromVersion: true,
                toVersion: true,
            },
        });

        relations.push(relation);
    }

    return relations;
}

/**
 * Helper to create bidirectional relations between resource versions
 */
export async function createBidirectionalRelations(
    prisma: any,
    versionId1: string,
    versionId2: string,
    forwardRelationType = "DEPENDENCY",
    backwardRelationType = "USED_BY",
    forwardLabels: string[] = ["dependency"],
    backwardLabels: string[] = ["used-by"],
) {
    const forwardRelation = await prisma.resource_version_relation.create({
        data: ResourceVersionRelationDbFactory.createComplete(
            versionId1,
            versionId2,
            forwardRelationType,
            forwardLabels,
        ),
        include: {
            fromVersion: true,
            toVersion: true,
        },
    });

    const backwardRelation = await prisma.resource_version_relation.create({
        data: ResourceVersionRelationDbFactory.createComplete(
            versionId2,
            versionId1,
            backwardRelationType,
            backwardLabels,
        ),
        include: {
            fromVersion: true,
            toVersion: true,
        },
    });

    return { forwardRelation, backwardRelation };
}
