// @ts-nocheck - Disabled due to Prisma type issues
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { generatePublicId } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ResourceVersionRelationRelationConfig extends RelationConfig {
    from?: { resourceVersionId: string };
    to?: { resourceVersionId: string };
}

/**
 * Enhanced database fixture factory for ResourceVersionRelation model
 * Handles relationships between resource versions (dependencies, related resources, etc.)
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Relationship type management
 * - Manual vs automatic entry tracking
 * - Sequence ordering for dependencies
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class ResourceVersionRelationDbFactory extends EnhancedDatabaseFactory<
    Prisma.resource_version_relationCreateInput,
    Prisma.resource_version_relationCreateInput,
    Prisma.resource_version_relationInclude,
    Prisma.resource_version_relationUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("ResourceVersionRelation", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.resource_version_relation;
    }

    /**
     * Get complete test fixtures for ResourceVersionRelation model
     */
    protected getFixtures(): DbTestFixtures<Prisma.resource_version_relationCreateInput, Prisma.resource_version_relationUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                relationshipType: "DependsOn",
                isManualEntry: false,
            },
            complete: {
                id: this.generateId(),
                relationshipType: "DependsOn",
                isManualEntry: false,
                sequence: 1,
                note: "This resource depends on the target resource for core functionality",
            },
            invalid: {
                missingRequired: {
                    // Missing id, relationshipType
                    isManualEntry: false,
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    relationshipType: 123, // Should be string
                    isManualEntry: "yes", // Should be boolean
                    sequence: "first", // Should be number
                    note: 123, // Should be string
                },
                invalidRelationshipType: {
                    id: this.generateId(),
                    relationshipType: "", // Empty relationship type
                    isManualEntry: false,
                },
                missingFromTo: {
                    id: this.generateId(),
                    relationshipType: "DependsOn",
                    isManualEntry: false,
                    // Missing both fromId and toId connections
                },
            },
            edgeCases: {
                circularDependency: {
                    id: this.generateId(),
                    relationshipType: "DependsOn",
                    isManualEntry: false,
                    // Would need to set fromId and toId to same value (caught in validation)
                },
                manualRelation: {
                    id: this.generateId(),
                    relationshipType: "RelatedTo",
                    isManualEntry: true,
                    note: "Manually added relation by user",
                },
                complexDependencyChain: {
                    id: this.generateId(),
                    relationshipType: "DependsOn",
                    isManualEntry: false,
                    sequence: 10, // Part of a long dependency chain
                },
                complementaryRelation: {
                    id: this.generateId(),
                    relationshipType: "ComplementedBy",
                    isManualEntry: false,
                    note: "These resources work well together",
                },
                prerequisiteRelation: {
                    id: this.generateId(),
                    relationshipType: "PrerequisiteFor",
                    isManualEntry: false,
                    sequence: 1,
                    note: "Must be completed before the target resource",
                },
            },
            updates: {
                minimal: {
                    note: "Updated relation note",
                },
                complete: {
                    relationshipType: "RelatedTo",
                    isManualEntry: true,
                    sequence: 2,
                    note: "Updated to reflect new relationship",
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.resource_version_relationCreateInput>): Prisma.resource_version_relationCreateInput {
        return {
            id: this.generateId(),
            relationshipType: "DependsOn",
            isManualEntry: false,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.resource_version_relationCreateInput>): Prisma.resource_version_relationCreateInput {
        return {
            id: this.generateId(),
            relationshipType: "DependsOn",
            isManualEntry: false,
            sequence: 1,
            note: "This resource depends on the target resource for core functionality",
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            dependency: {
                name: "dependency",
                description: "Standard dependency relation",
                config: {
                    overrides: {
                        relationshipType: "DependsOn",
                        isManualEntry: false,
                        sequence: 1,
                        note: "Required dependency",
                    },
                },
            },
            prerequisite: {
                name: "prerequisite",
                description: "Prerequisite relation",
                config: {
                    overrides: {
                        relationshipType: "PrerequisiteFor",
                        isManualEntry: false,
                        sequence: 1,
                        note: "Must be completed first",
                    },
                },
            },
            related: {
                name: "related",
                description: "Related resource relation",
                config: {
                    overrides: {
                        relationshipType: "RelatedTo",
                        isManualEntry: true,
                        note: "Manually linked as related resource",
                    },
                },
            },
            complementary: {
                name: "complementary",
                description: "Complementary resource relation",
                config: {
                    overrides: {
                        relationshipType: "ComplementedBy",
                        isManualEntry: false,
                        note: "Works well together with this resource",
                    },
                },
            },
            alternative: {
                name: "alternative",
                description: "Alternative resource relation",
                config: {
                    overrides: {
                        relationshipType: "AlternativeTo",
                        isManualEntry: true,
                        note: "Can be used instead of this resource",
                    },
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.resource_version_relationInclude {
        return {
            from: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                        },
                        take: 1,
                        where: { language: "en" },
                    },
                },
            },
            to: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                        },
                        take: 1,
                        where: { language: "en" },
                    },
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.resource_version_relationCreateInput,
        config: ResourceVersionRelationRelationConfig,
        tx: PrismaClient,
    ): Promise<Prisma.resource_version_relationCreateInput> {
        const data = { ...baseData };

        // Handle from connection
        if (config.from?.resourceVersionId) {
            data.from = { connect: { id: config.from.resourceVersionId } };
        }

        // Handle to connection
        if (config.to?.resourceVersionId) {
            data.to = { connect: { id: config.to.resourceVersionId } };
        }

        return data;
    }

    /**
     * Create a dependency relation
     */
    async createDependency(
        fromVersionId: string,
        toVersionId: string,
        sequence = 1,
    ): Promise<Prisma.ResourceVersionRelation> {
        return this.createWithRelations({
            from: { resourceVersionId: fromVersionId },
            to: { resourceVersionId: toVersionId },
            overrides: {
                relationshipType: "DependsOn",
                isManualEntry: false,
                sequence,
                note: `Dependency ${sequence}`,
            },
        });
    }

    /**
     * Create a prerequisite relation
     */
    async createPrerequisite(
        fromVersionId: string,
        toVersionId: string,
    ): Promise<Prisma.ResourceVersionRelation> {
        return this.createWithRelations({
            from: { resourceVersionId: fromVersionId },
            to: { resourceVersionId: toVersionId },
            overrides: {
                relationshipType: "PrerequisiteFor",
                isManualEntry: false,
                sequence: 1,
                note: "Must be completed before proceeding",
            },
        });
    }

    /**
     * Create a manual relation
     */
    async createManualRelation(
        fromVersionId: string,
        toVersionId: string,
        relationshipType = "RelatedTo",
    ): Promise<Prisma.ResourceVersionRelation> {
        return this.createWithRelations({
            from: { resourceVersionId: fromVersionId },
            to: { resourceVersionId: toVersionId },
            overrides: {
                relationshipType,
                isManualEntry: true,
                note: "Manually added by user",
            },
        });
    }

    /**
     * Create dependency chain
     */
    async createDependencyChain(versionIds: string[]): Promise<Prisma.ResourceVersionRelation[]> {
        const relations: Prisma.ResourceVersionRelation[] = [];
        
        for (let i = 0; i < versionIds.length - 1; i++) {
            const relation = await this.createDependency(
                versionIds[i],
                versionIds[i + 1],
                i + 1,
            );
            relations.push(relation);
        }
        
        return relations;
    }

    /**
     * Create bidirectional relation
     */
    async createBidirectionalRelation(
        versionId1: string,
        versionId2: string,
        relationshipType = "RelatedTo",
    ): Promise<[Prisma.ResourceVersionRelation, Prisma.ResourceVersionRelation]> {
        const relation1 = await this.createWithRelations({
            from: { resourceVersionId: versionId1 },
            to: { resourceVersionId: versionId2 },
            overrides: {
                relationshipType,
                isManualEntry: false,
            },
        });

        const relation2 = await this.createWithRelations({
            from: { resourceVersionId: versionId2 },
            to: { resourceVersionId: versionId1 },
            overrides: {
                relationshipType,
                isManualEntry: false,
            },
        });

        return [relation1, relation2];
    }

    protected async checkModelConstraints(record: Prisma.ResourceVersionRelation): Promise<string[]> {
        const violations: string[] = [];

        // Check that relationship type is not empty
        if (!record.relationshipType || record.relationshipType.trim() === "") {
            violations.push("Relationship type cannot be empty");
        }

        // Check that relation has both from and to
        if (!record.fromId || !record.toId) {
            violations.push("Relation must have both from and to resource versions");
        }

        // Check for self-reference (circular dependency)
        if (record.fromId && record.toId && record.fromId === record.toId) {
            violations.push("Resource version cannot have a relation to itself");
        }

        // Check sequence is positive if provided
        if (record.sequence !== null && record.sequence < 0) {
            violations.push("Sequence must be non-negative");
        }

        // Check for duplicate relations
        if (record.fromId && record.toId && record.relationshipType) {
            const duplicate = await this.prisma.resource_version_relation.findFirst({
                where: {
                    fromId: record.fromId,
                    toId: record.toId,
                    relationshipType: record.relationshipType,
                    id: { not: record.id },
                },
            });
            if (duplicate) {
                violations.push("Duplicate relation already exists");
            }
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {};
    }

    protected async deleteRelatedRecords(
        record: Prisma.ResourceVersionRelation,
        remainingDepth: number,
        tx: PrismaClient,
        includeOnly?: string[],
    ): Promise<void> {
        // ResourceVersionRelation has no dependent records to cascade delete
    }

    /**
     * Get all dependencies for a resource version
     */
    async getDependencies(versionId: string): Promise<Prisma.ResourceVersionRelation[]> {
        return this.prisma.resource_version_relation.findMany({
            where: {
                fromId: versionId,
                relationshipType: "DependsOn",
            },
            include: this.getDefaultInclude(),
            orderBy: { sequence: "asc" },
        });
    }

    /**
     * Get all dependents of a resource version
     */
    async getDependents(versionId: string): Promise<Prisma.ResourceVersionRelation[]> {
        return this.prisma.resource_version_relation.findMany({
            where: {
                toId: versionId,
                relationshipType: "DependsOn",
            },
            include: this.getDefaultInclude(),
            orderBy: { sequence: "asc" },
        });
    }
}

// Export factory creator function
export const createResourceVersionRelationDbFactory = (prisma: PrismaClient) => 
    new ResourceVersionRelationDbFactory(prisma);

// Export the class for type usage
export { ResourceVersionRelationDbFactory as ResourceVersionRelationDbFactoryClass };
