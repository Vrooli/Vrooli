/**
 * Routine Database Factory
 * 
 * Creates routine records in the test database using resource/resource_version pattern
 */

import { generatePK, generatePublicId, ResourceType } from "@vrooli/shared";
import { DbProvider } from "../../../../db/provider.js";
import type { RoutineTestData } from "../../types.js";

export class RoutineDbFactory {
    async create(routineData: RoutineTestData): Promise<RoutineTestData> {
        const userId = routineData.created_by || generatePK();
        
        // First ensure the user exists
        const existingUser = await DbProvider.get().user.findUnique({
            where: { id: BigInt(userId) },
        });
        
        if (!existingUser) {
            await DbProvider.get().user.create({
                data: {
                    id: BigInt(userId),
                    publicId: generatePublicId(),
                    name: "Test User",
                    isPrivate: false,
                    isBot: false,
                    profileImage: null,
                },
            });
        }
        
        // Create routine as resource with resourceType: "Routine"
        const resource = await DbProvider.get().resource.create({
            data: {
                id: BigInt(routineData.id),
                publicId: generatePublicId(),
                resourceType: ResourceType.Routine,
                isPrivate: false,
                hasCompleteVersion: true,
                isDeleted: false,
                isInternal: false,
                completedAt: new Date(),
                ownedByUserId: BigInt(userId),
            },
        });

        // Create a resource version for the routine
        const version = routineData.versions[0];
        if (!version) {
            throw new Error("Routine must have at least one version");
        }
        
        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: BigInt(version.id),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionLabel: routineData.versions[0]?.version || "1.0.0",
                versionIndex: 0,
                isComplete: true,
                isLatest: true,
                isPrivate: false,
                complexity: routineData.versions[0]?.complexity || 1,
                isAutomatable: true,
                translations: {
                    create: {
                        id: generatePK(),
                        language: "en",
                        name: routineData.name,
                        description: routineData.description,
                    },
                },
            },
            include: {
                translations: true,
            },
        });

        // Return the test data structure with database ids
        return {
            ...routineData,
            id: resource.id.toString(),
            versions: [{
                ...version,
                id: resourceVersion.id.toString(),
            }],
        };
    }

    async createBatch(routines: RoutineTestData[]): Promise<any[]> {
        return Promise.all(routines.map(r => this.create(r)));
    }

    async update(id: string, data: Partial<RoutineTestData>): Promise<any> {
        return DbProvider.get().resource.update({
            where: { id: BigInt(id) },
            data: {
                translations: data.name || data.description ? {
                    update: {
                        where: { language: "en" },
                        data: {
                            name: data.name,
                            description: data.description,
                        },
                    },
                } : undefined,
            },
        });
    }

    async delete(id: string): Promise<void> {
        await DbProvider.get().resource.delete({
            where: { id: BigInt(id) },
        });
    }
}
