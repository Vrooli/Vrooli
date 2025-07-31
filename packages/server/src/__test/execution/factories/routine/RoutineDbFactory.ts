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
        const userId = routineData.created_by || generatePK().toString();

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
        // First update the resource if needed
        const resource = await DbProvider.get().resource.update({
            where: { id: BigInt(id) },
            data: {
                // Resource doesn't have many fields that can be updated from RoutineTestData
                // Most routine-specific data is in the version
            },
        });

        // If name or description are being updated, update the resource_version's translations
        if (data.name || data.description) {
            // Find the latest version for this resource
            const latestVersion = await DbProvider.get().resource_version.findFirst({
                where: {
                    rootId: BigInt(id),
                    isLatest: true,
                },
                include: {
                    translations: true,
                },
            });

            if (latestVersion) {
                // Update the translation
                await DbProvider.get().resource_translation.updateMany({
                    where: {
                        resourceVersionId: latestVersion.id,
                        language: "en",
                    },
                    data: {
                        ...(data.name && { name: data.name }),
                        ...(data.description && { description: data.description }),
                    },
                });
            }
        }

        return resource;
    }

    async delete(id: string): Promise<void> {
        await DbProvider.get().resource.delete({
            where: { id: BigInt(id) },
        });
    }
}
