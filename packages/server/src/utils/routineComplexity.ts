import { ResourceSubType } from "@local/shared";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";

export async function calculateComplexity(
    inputs: {
        id: string,
        resourceSubType?: string | null,
        relatedVersionsCreate?: { toVersionConnect: string }[] | null,
        relatedVersionsDisconnect?: string[] | null,
    }[],
    disallowIds: string[],
): Promise<Record<string, number>> {
    const inputIds = inputs.map(i => i.id);
    if (inputIds.some(id => disallowIds.includes(id))) {
        throw new CustomError("0370", "InvalidArgs", { detail: "Circular dependency detected." });
    }

    // Create a set of all IDs that we need to fetch.
    // This includes the inputs, as well as any related versions that are being created/updated/deleted.
    const fetchIds = new Set<string>(inputIds);
    for (const input of inputs) {
        if (input.relatedVersionsCreate) {
            for (const rel of input.relatedVersionsCreate) {
                fetchIds.add(rel.toVersionConnect);
            }
        }
        if (input.relatedVersionsDisconnect) {
            for (const rel of input.relatedVersionsDisconnect) {
                fetchIds.add(rel);
            }
        }
    }

    // Find inputs that already exist in the database, and every subroutine they depend on.
    const fetchedData = await DbProvider.get().resource_version.findMany({
        where: { id: { in: Array.from(fetchIds).map(i => BigInt(i)) } },
        select: {
            id: true,
            complexity: true,
            config: true,
            resourceSubType: true,
            root: {
                select: {
                    resourceType: true,
                },
            },
            relatedVersions: {
                select: {
                    labels: true,
                    toVersion: {
                        select: {
                            id: true,
                            complexity: true,
                            resourceSubType: true,
                        },
                    },
                },
            },
        },
    });

    // Create a map of all resource versions (including related versions) by ID, so we can reference them later.
    const resourceVersionsById = new Map<string, { complexity: number | null }>(
        fetchedData.map(rv => [rv.id.toString(), rv]),
    );
    for (const rv of fetchedData) {
        if (rv.relatedVersions) {
            for (const rel of rv.relatedVersions) {
                resourceVersionsById.set(rel.toVersion.id.toString(), { complexity: rel.toVersion.complexity });
            }
        }
    }

    // Initialize result
    const complexityById: Record<string, number> = {};

    // Process inputs
    for (const input of inputs) {
        // Get the existing data
        const existingData = resourceVersionsById.get(input.id);

        // If the input is not a routine, return 0
        const resourceSubType = input.resourceSubType ?? existingData?.resourceSubType;
        if (!resourceSubType?.startsWith("Routine")) {
            complexityById[input.id] = 0;
            continue;
        }
        // If the input is not a multi-step routine, return 0
        if (resourceSubType !== ResourceSubType.RoutineMultiStep) {
            complexityById[input.id] = 0;
            continue;
        }

        // If it doesn't exist (i.e. it's a new routine), sum all related versions that are routines
        if (!existingData) {
            // Find all related version in the resourceVersionsById map. Then filter out any that are not routines.
            const relatedRoutineVersionsSum = input.relatedVersionsCreate?.map(rv => resourceVersionsById.get(rv.toVersionConnect))
                .filter(rv => rv?.complexity !== null)
                .map(rv => rv!.complexity)
                .reduce((a, b) => a! + b!, 0);
            complexityById[input.id] = relatedRoutineVersionsSum ?? 0;
        }
        // If it does exist, do (existingComplexity + sum of new related routines - sum of deleted related routines)
        else {
            const relatedRoutineVersionsSum = input.relatedVersionsCreate?.map(rv => resourceVersionsById.get(rv.toVersionConnect))
                .filter(rv => rv?.complexity !== null)
                .map(rv => rv!.complexity)
                .reduce((a, b) => a! + b!, 0);
            complexityById[input.id] = (existingData.complexity ?? 0) + (relatedRoutineVersionsSum ?? 0) - (input.relatedVersionsDisconnect?.map(rv => resourceVersionsById.get(rv))
                .filter(rv => rv?.complexity !== null)
                .map(rv => rv!.complexity)
                .reduce((a, b) => a! + b!, 0) ?? 0);
        }
    }

    // Return the result
    return complexityById;
}
