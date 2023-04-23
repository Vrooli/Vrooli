import { onlyValidIds } from "./onlyValidIds";
export async function existsArray({ ids, prismaDelegate, where }) {
    if (ids.length === 0)
        return [];
    const idsToQuery = onlyValidIds(ids);
    const objects = await prismaDelegate.findMany({
        where,
        select: { id: true },
    });
    return idsToQuery.map(id => objects.some(({ id: objectId }) => objectId === id));
}
//# sourceMappingURL=existsArray.js.map