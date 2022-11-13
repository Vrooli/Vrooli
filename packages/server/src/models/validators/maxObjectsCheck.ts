/**
 * This file handles checking if a cud action violates any max object limits. 
 * Limits are defined both on the user and organizational level, and can be raised 
 * by the standard premium subscription, or a custom subscription.
 * 
 * The general idea is that users can have a limited number of private objects, and
 * a larger number of public objects. Organizations can have even more private objects, and 
 * even more public objects. The limits should be just low enough to encourage people to make public 
 * objects and transfer them to organizations, but not so low that it's impossible to use the platform.
 * 
 * We want objects to be owned by organizations rather than users, as this means the objects are tied to 
 * the organization's governance structure.
 */
import { CODE } from "@shared/consts";
import { CustomError, genErrorCode } from "../../events";
import { PrismaDelegate, PrismaType } from "../../types";
import { ObjectMap } from "../builder";
import { GraphQLModelType, Validator } from "../types";
import { getValidatorAndDelegate } from "../utils";
import { MaxObjectsCheckProps } from "./types";

/**
 * Map which defines the maximum number of private objects a user can have. Other maximum checks 
 * (e.g. maximum nodes in a routine, projects in an organization) must be checked elsewhere.
 */
const maxPrivateObjectsUserMap: { [key in GraphQLModelType]?: (hasPremium: boolean) => number } = {
    // Api: (hp) => hp ? 5 : 1,
    Comment: () => 0, // No private comments
    Email: () => 5, // Can only be applied to user, and is always private. So it doesn't show up in the other maps
    // Note: (hp) => hp ? 1000 : 25,
    Organization: () => 1,
    Project: (hp) => hp ? 100 : 3,
    Routine: (hp) => hp ? 250 : 25,
    // SmartContract: (hp) => hp ? 25 : 1,
    // ScheduleFilter: (hp) => hp ? 100 : 10, // *Per user schedule. Doesn't show up in the other maps
    Standard: (hp) => hp ? 25 : 3,
    Wallet: (hp) => hp ? 5 : 1, // Is always private. So it doesn't show up in public maps
}

/**
 * Map which defines the maximum number of public objects a user can have. Other maximum checks must be checked elsewhere.
 */
const maxPublicObjectsUserMap: { [key in GraphQLModelType]?: (hasPremium: boolean) => number } = {
    // Api: (hp) => hp ? 50 : 10,
    Comment: () => 10000,
    Handle: () => 1, // Is always public. So it doesn't show up in private maps
    // Note: (hp) => hp ? 1000 : 25,
    Organization: (hp) => hp ? 25 : 3,
    Project: (hp) => hp ? 250 : 25,
    Routine: (hp) => hp ? 1000 : 100,
    // SmartContract: (hp) => hp ? 250 : 25,
    Standard: (hp) => hp ? 1000 : 100,
}

/**
 * Map which defines the maximum number of private objects an organization can have. Other maximum checks must be checked elsewhere.
 */
const maxPrivateObjectsOrganizationMap: { [key in GraphQLModelType]?: (hasPremium: boolean) => number } = {
    // Api: (hp) => hp ? 50 : 10,
    // Note: (hp) => hp ? 1000 : 25,
    Project: (hp) => hp ? 100 : 3,
    Routine: (hp) => hp ? 250 : 25,
    // SmartContract: (hp) => hp ? 25 : 1,
    Standard: (hp) => hp ? 25 : 3,
    Wallet: (hp) => hp ? 5 : 1, // Is always private. So it doesn't show up in public maps
}

/**
 * Map which defines the maximum number of public objects an organization can have. Other maximum checks must be checked elsewhere.
 */
const maxPublicObjectsOrganizationMap: { [key in GraphQLModelType]?: (hasPremium: boolean) => number } = {
    // Api: (hp) => hp ? 50 : 10,
    Handle: () => 1, // Is always public. So it doesn't show up in private maps
    // Note: (hp) => hp ? 1000 : 25,
    Project: (hp) => hp ? 250 : 25,
    Routine: (hp) => hp ? 1000 : 100,
    // SmartContract: (hp) => hp ? 250 : 25,
    Standard: (hp) => hp ? 1000 : 100,
}

type MaxObjectsCheckData = {
    id: string,
    createdByUserId?: string | null | undefined,
    createdByOrganizationId?: string | null | undefined,
    userId?: string | null | undefined,
    organizationId?: string | null | undefined,
}

/**
 * Validates that creating a new project, routine, or standard will not exceed the user's limit. Checks both 
 * for personal limits and organizational limits, factoring in the user or organization's premium status.
 */
export async function maxObjectsCheck<GraphQLCreate extends { [x: string]: any }, GraphQLUpdate extends { [x: string]: any }>({
    createMany,
    deleteMany,
    objectType,
    prisma,
    updateMany,
    userId,
}: MaxObjectsCheckProps<GraphQLCreate, GraphQLUpdate>) {
    // Initialize counts for user and organization
    let totalUserIdCount = 0; // Queries will only return objects which belong to this user, or belong to an organization (though not necessarily one this user is a member of)
    let totalOrganizationIds: { [id: string]: number } = {}; // All returned organizations will be counted, even if the user is not a member of them. Just makes the logic easier
    // Find validator and prisma delegate for this object type
    const { validator, prismaDelegate } = getValidatorAndDelegate(objectType, prisma, 'maxObjectsCheck');
    // Add createMany to counts. Every createMany that doesn't contain a createdByOrganizationId or organizationId is assumed to belong to the user instead
    //TODO
    // Use prisma and validator to query for all updateMany and deleteMany objects, returning the owner of each object
    //TODO
    // Separate updateMany and deleteMany results. Add updateMany owner ids to the counts, and remove deleteMany owner ids from the counts
    //TODO
    // If counts of userId or any organizationId are greater than the maximum allowed, throw an error
    //TODO

    let temp = await prisma.routine_version.findUnique({
        where: { id: 'asdfa'},
        select: {
            root: { 
                select: {
                    user: {
                        select: {
                            id: true,
                        }
                    },
                    organization: {
                        select: {
                            id: true,
                            permissions: true,
                        }
                    },
                    permissions: true,

                }
            }
        }
    })

    /**
     * Helper for converting an array of strings to a map of occurence counts
     */
    const countHelper = (arr: (string | null | undefined)[]): { [id: string]: number } => {
        const result: { [id: string]: number } = {};
        arr.forEach(x => {
            if (!x) return;
            result[x] = (result[x] ?? 0) + 1;
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