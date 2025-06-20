import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface ResourceVersionRelationRelationConfig extends RelationConfig {
    resourceVersion: { id: string };
    target: {
        id: string;
        type: "ProjectVersion" | "RoutineVersion" | "SmartContractVersion" | "DataStructureVersion";
    };
    index?: number;
}

/**
 * Database fixture factory for ResourceVersionRelation model
 * Handles junction table connections between ResourceVersion and other versioned objects
 */
export class ResourceVersionRelationDbFactory extends DatabaseFixtureFactory<
    Prisma.ResourceVersionRelation,
    Prisma.ResourceVersionRelationCreateInput,
    Prisma.ResourceVersionRelationInclude,
    Prisma.ResourceVersionRelationUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('ResourceVersionRelation', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.resourceVersionRelation;
    }

    protected getMinimalData(overrides?: Partial<Prisma.ResourceVersionRelationCreateInput>): Prisma.ResourceVersionRelationCreateInput {
        return {
            id: generatePK(),
            index: 0,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.ResourceVersionRelationCreateInput>): Prisma.ResourceVersionRelationCreateInput {
        return {
            id: generatePK(),
            index: 0,
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.ResourceVersionRelationInclude {
        return {
            resourceVersion: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                            description: true,
                        },
                    },
                    root: {
                        select: {
                            id: true,
                            publicId: true,
                            resourceType: true,
                        },
                    },
                },
            },
            projectVersion: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                            description: true,
                        },
                    },
                },
            },
            routineVersion: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                            description: true,
                        },
                    },
                },
            },
            smartContractVersion: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                            description: true,
                        },
                    },
                },
            },
            dataStructureVersion: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                            description: true,
                        },
                    },
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.ResourceVersionRelationCreateInput,
        config: ResourceVersionRelationRelationConfig,
        tx: any
    ): Promise<Prisma.ResourceVersionRelationCreateInput> {
        let data = { ...baseData };

        // Handle resource version connection
        if (config.resourceVersion) {
            data.resourceVersion = { connect: { id: config.resourceVersion.id } };
        }

        // Handle target connection based on type
        if (config.target) {
            data.index = config.index ?? 0;
            
            switch (config.target.type) {
                case "ProjectVersion":
                    data.projectVersion = { connect: { id: config.target.id } };
                    break;
                case "RoutineVersion":
                    data.routineVersion = { connect: { id: config.target.id } };
                    break;
                case "SmartContractVersion":
                    data.smartContractVersion = { connect: { id: config.target.id } };
                    break;
                case "DataStructureVersion":
                    data.dataStructureVersion = { connect: { id: config.target.id } };
                    break;
            }
        }

        return data;
    }

    /**
     * Create relation between resource version and project version
     */
    async createResourceToProject(
        resourceVersionId: string,
        projectVersionId: string,
        index: number = 0
    ): Promise<Prisma.ResourceVersionRelation> {
        return this.createWithRelations({
            resourceVersion: { id: resourceVersionId },
            target: { id: projectVersionId, type: "ProjectVersion" },
            index,
        });
    }

    /**
     * Create relation between resource version and routine version
     */
    async createResourceToRoutine(
        resourceVersionId: string,
        routineVersionId: string,
        index: number = 0
    ): Promise<Prisma.ResourceVersionRelation> {
        return this.createWithRelations({
            resourceVersion: { id: resourceVersionId },
            target: { id: routineVersionId, type: "RoutineVersion" },
            index,
        });
    }

    /**
     * Create relation between resource version and smart contract version
     */
    async createResourceToSmartContract(
        resourceVersionId: string,
        smartContractVersionId: string,
        index: number = 0
    ): Promise<Prisma.ResourceVersionRelation> {
        return this.createWithRelations({
            resourceVersion: { id: resourceVersionId },
            target: { id: smartContractVersionId, type: "SmartContractVersion" },
            index,
        });
    }

    /**
     * Create relation between resource version and data structure version
     */
    async createResourceToDataStructure(
        resourceVersionId: string,
        dataStructureVersionId: string,
        index: number = 0
    ): Promise<Prisma.ResourceVersionRelation> {
        return this.createWithRelations({
            resourceVersion: { id: resourceVersionId },
            target: { id: dataStructureVersionId, type: "DataStructureVersion" },
            index,
        });
    }

    /**
     * Create multiple relations for a resource to different targets
     */
    async createMultipleRelations(
        resourceVersionId: string,
        targets: Array<{ id: string; type: "ProjectVersion" | "RoutineVersion" | "SmartContractVersion" | "DataStructureVersion" }>
    ): Promise<Prisma.ResourceVersionRelation[]> {
        const relations: Prisma.ResourceVersionRelation[] = [];

        for (let i = 0; i < targets.length; i++) {
            const target = targets[i];
            const relation = await this.createWithRelations({
                resourceVersion: { id: resourceVersionId },
                target: { id: target.id, type: target.type },
                index: i,
            });
            relations.push(relation);
        }

        return relations;
    }

    /**
     * Create ordered relations (useful for documentation sequences)
     */
    async createOrderedRelations(
        resourceVersionId: string,
        targetIds: string[],
        targetType: "ProjectVersion" | "RoutineVersion" | "SmartContractVersion" | "DataStructureVersion"
    ): Promise<Prisma.ResourceVersionRelation[]> {
        const relations: Prisma.ResourceVersionRelation[] = [];

        for (let i = 0; i < targetIds.length; i++) {
            const relation = await this.createWithRelations({
                resourceVersion: { id: resourceVersionId },
                target: { id: targetIds[i], type: targetType },
                index: i,
            });
            relations.push(relation);
        }

        return relations;
    }

    protected async checkModelConstraints(record: Prisma.ResourceVersionRelation): Promise<string[]> {
        const violations: string[] = [];

        // Check that exactly one target type is connected
        const targetCount = [
            record.projectVersionId,
            record.routineVersionId,
            record.smartContractVersionId,
            record.dataStructureVersionId,
        ].filter(id => id !== null && id !== undefined).length;

        if (targetCount !== 1) {
            violations.push('ResourceVersionRelation must connect to exactly one target type');
        }

        // Check index is non-negative
        if (record.index < 0) {
            violations.push('Index must be non-negative');
        }

        // Check for duplicate index within the same resource version
        if (record.resourceVersionId) {
            const duplicateIndex = await this.prisma.resourceVersionRelation.count({
                where: {
                    resourceVersionId: record.resourceVersionId,
                    index: record.index,
                    id: { not: record.id },
                },
            });
            
            if (duplicateIndex > 0) {
                violations.push('Index must be unique within the same resource version');
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
                // Missing resourceVersionId and target
                index: 0,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                index: "zero", // Should be number
                resourceVersionId: 123, // Should be string
            },
            noTarget: {
                id: generatePK(),
                index: 0,
                resourceVersionId: "resource_version_123",
                // Missing target connection
            },
            multipleTargets: {
                id: generatePK(),
                index: 0,
                resourceVersionId: "resource_version_123",
                projectVersionId: "project_version_123",
                routineVersionId: "routine_version_123", // Can't have multiple targets
            },
            negativeIndex: {
                id: generatePK(),
                index: -1, // Invalid negative index
                resourceVersionId: "resource_version_123",
                projectVersionId: "project_version_123",
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.ResourceVersionRelationCreateInput> {
        return {
            highIndex: {
                ...this.getMinimalData(),
                index: 999,
            },
            zeroIndex: {
                ...this.getMinimalData(),
                index: 0,
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            resourceVersion: true,
            projectVersion: true,
            routineVersion: true,
            smartContractVersion: true,
            dataStructureVersion: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.ResourceVersionRelation,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Junction table - no cascading deletes needed
        // The relation itself will be deleted by the parent deleteRelatedRecords call
    }

    /**
     * Bulk create relations for a resource version
     */
    async bulkCreateRelations(
        resourceVersionId: string,
        relations: Array<{
            targetId: string;
            targetType: "ProjectVersion" | "RoutineVersion" | "SmartContractVersion" | "DataStructureVersion";
            index?: number;
        }>
    ): Promise<Prisma.ResourceVersionRelation[]> {
        const results: Prisma.ResourceVersionRelation[] = [];

        // Sort by index to ensure proper ordering
        const sortedRelations = relations.sort((a, b) => (a.index || 0) - (b.index || 0));

        for (const relation of sortedRelations) {
            const created = await this.createWithRelations({
                resourceVersion: { id: resourceVersionId },
                target: { id: relation.targetId, type: relation.targetType },
                index: relation.index ?? results.length,
            });
            results.push(created);
        }

        return results;
    }

    /**
     * Get relations by resource version with ordering
     */
    async getRelationsByResourceVersion(resourceVersionId: string): Promise<Prisma.ResourceVersionRelation[]> {
        return this.prisma.resourceVersionRelation.findMany({
            where: { resourceVersionId },
            orderBy: { index: 'asc' },
            include: this.getDefaultInclude(),
        });
    }

    /**
     * Reorder relations for a resource version
     */
    async reorderRelations(
        resourceVersionId: string,
        newOrder: Array<{ relationId: string; newIndex: number }>
    ): Promise<void> {
        await this.prisma.$transaction(async (tx) => {
            for (const { relationId, newIndex } of newOrder) {
                await tx.resourceVersionRelation.update({
                    where: { id: relationId },
                    data: { index: newIndex },
                });
            }
        });
    }
}

// Export factory creator function
export const createResourceVersionRelationDbFactory = (prisma: PrismaClient) => new ResourceVersionRelationDbFactory(prisma);