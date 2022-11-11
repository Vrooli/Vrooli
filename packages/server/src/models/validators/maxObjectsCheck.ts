import { MaxObjectsCheckProps } from "./types";

type MaxObjectsCheckData = {
    id: string,
    createdByUserId?: string | null | undefined,
    createdByOrganizationId?: string | null | undefined,
    userId?: string | null | undefined,
    organizationId?: string | null | undefined,
}

/**
 * Validates that creating a new project, routine, or standard will not exceed the user's limit
 */
export async function maxObjectsCheck<GraphQLCreate extends { [x: string]: any }, GraphQLUpdate extends { [x: string]: any }>({
    createMany,
    deleteMany,
    objectType,
    prisma,
    updateMany,
    userId,
}: MaxObjectsCheckProps<GraphQLCreate, GraphQLUpdate>) {
    let totalUserIdCount = 0;
    let totalOrganizationIds: { [id: string]: number } = {};
    /**
     * Helper for converting an array of strings to a map of occurence counts
     */
    const countHelper = (arr: (string | null | undefined)[]): { [id: string]: number } => {
        const result: { [id: string]: number } = {};
        arr.forEach(x => {
            if (!x) return;
            if (result[x]) {
                result[x] += 1;
            }
            else {
                result[x] = 1;
            }
        });
        return result;
    }
    /**
     * Helper for adding counts to the total counts
     */
    const addToCounts = (data: MaxObjectsCheckData[]) => {
        totalUserIdCount += data.filter(x => x.createdByUserId === userId || x.userId === userId).length;
        const organizationIds = countHelper(data.map(x => x.createdByOrganizationId || x.organizationId));
        Object.keys(organizationIds).forEach(id => {
            if (totalOrganizationIds[id]) {
                totalOrganizationIds[id] += organizationIds[id];
            }
            else {
                totalOrganizationIds[id] = organizationIds[id];
            }
        });
    }
    /**
     * Helper for removing queried counts from the total counts
     */
    const removeFromCounts = (userCounts: number, organizationCounts: { [id: string]: number }) => {
        totalUserIdCount -= userCounts;
        Object.keys(organizationCounts).forEach(id => {
            if (totalOrganizationIds[id]) {
                totalOrganizationIds[id] -= organizationCounts[id];
            }
        });
    }
    /**
     * Helper for querying existing data
     * @returns Count of userId, and count of organizationId by ID
     */
    const queryExisting = async (ids: string[]): Promise<[number, { [id: string]: number }]> => {
        let userIdCount: number;
        let organizationIds: { [id: string]: number };
        if (objectType === 'Project') {
            const objects = await prisma.project.findMany({
                where: { id: { in: ids } },
                select: { id: true, userId: true, organizationId: true },
            });
            userIdCount = objects.filter(x => x.userId === userId).length;
            organizationIds = countHelper(objects.map(x => x.organizationId));
        }
        else if (objectType === 'Routine') {
            const objects = await prisma.routine.findMany({
                where: { id: { in: ids } },
                select: { id: true, userId: true, organizationId: true },
            });
            userIdCount = objects.filter(x => x.userId === userId).length;
            organizationIds = countHelper(objects.map(x => x.organizationId));
        }
        else {
            const objects = await prisma.standard.findMany({
                where: { id: { in: ids } },
                select: { id: true, createdByUserId: true, createdByOrganizationId: true },
            });
            userIdCount = objects.filter(x => x.createdByUserId === userId).length;
            organizationIds = countHelper(objects.map(x => x.createdByOrganizationId));
        }
        return [userIdCount, organizationIds];
    }
    // Add IDs in createMany to total counts
    if (createMany) {
        addToCounts(createMany);
    }
    // Add new IDs in updateMany to total counts, and remove existing IDs from total counts
    if (updateMany) {
        const newObjects = updateMany.map(u => u.data);
        addToCounts(newObjects);
        const [userIdCount, organizationIds] = await queryExisting(updateMany.map(u => u.where.id));
        removeFromCounts(userIdCount, organizationIds);
    }
    // Remove IDs in deleteMany from total counts
    if (deleteMany) {
        const [userIdCount, organizationIds] = await queryExisting(deleteMany);
        removeFromCounts(userIdCount, organizationIds);
    }
    // If the total counts exceed the max, throw an error
    if (totalUserIdCount > maxCount) {
        throw new CustomError(CODE.Unauthorized, `You have reached the maximum number of ${objectType}s you can create on this account.`, { code: genErrorCode('0260') });
    }
    if (Object.keys(totalOrganizationIds).some(id => totalOrganizationIds[id] > maxCount)) {
        throw new CustomError(CODE.Unauthorized, `You have reached the maximum number of ${objectType}s you can create on this organization.`, { code: genErrorCode('0261') });
    }
}