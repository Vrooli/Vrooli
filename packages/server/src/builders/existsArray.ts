import { onlyValidIds } from "./onlyValidIds";
import { ExistsArrayProps } from "./types";

/**
 * Helper function for querying a list of objects using a specified search query
 * @returns Array in the same order as the ids, with a boolean indicating whether the object was found
 */
export async function existsArray({ ids, prismaDelegate, where }: ExistsArrayProps): Promise<Array<boolean>> {
    if (ids.length === 0) return [];
    // Take out nulls
    const idsToQuery = onlyValidIds(ids);
    // Query
    const objects = await prismaDelegate.findMany({
        where,
        select: { id: true },
    })
    // Convert to array of booleans
    return idsToQuery.map(id => objects.some(({ id: objectId }: { id: string }) => objectId === id));
}