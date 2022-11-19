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
import { SessionUser } from "../../schema/types";
import { PrismaType } from "../../types";
import { GraphQLModelType } from "../types";
import { getValidator } from "../utils";
import { QueryAction } from "../utils/types";

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

/**
 * Validates that the user will not exceed any maximum object limits after the given action. Checks both 
 * for personal limits and organizational limits, factoring in the user or organization's premium status. 
 * Throws an error if a limit will be exceeded.
 * @param authDataById Map of all queried data required to validate permissions, keyed by ID.
 * @param idsByAction Map of object IDs to validate permissions for, keyed by action. We store actions this way (instead of keyed by ID) 
 * in case one ID is used for multiple actions.
 * @parma userId ID of user requesting permissions
 */
export async function maxObjectsCheck(
    authDataById: { [id: string]: { __typename: GraphQLModelType, [x: string]: any } },
    idsByAction: { [key in QueryAction]?: string[] },
    prisma: PrismaType,
    userData: SessionUser,
) {
    // Initialize counts. This is used to count how many objects a user or organization will have after every action is applied.
    const counts: { [key in GraphQLModelType]?: { [ownerId: string]: number } } = {}
    // Loop through every "Create" action, and increment the count for the object type
    if (idsByAction.Create) {
        for (const id of idsByAction.Create) {
            // Get auth data
            const authData = authDataById[id]
            // Get validator
            const validator = getValidator(authData.__typename, 'maxObjectsCheck-create')
            // Find owner and object type
            const owners = validator.owner(authData);
            // Increment count for owner
            const ownerId: string | undefined = owners.Organization?.id ?? owners.User?.id;
            if (!ownerId) throw new CustomError(CODE.InternalError, 'Could not find owner ID for object', { code: genErrorCode('0310') });
            counts[authData.__typename] = counts[authData.__typename] || {}
            counts[authData.__typename]![ownerId] = (counts[authData.__typename]![ownerId] || 0) + 1
        }
    }
    // Loop through every "Delete" action, and decrement the count for the object type
    if (idsByAction.Delete) {
        for (const id of idsByAction.Delete) {
            // Get auth data
            const authData = authDataById[id]
            // Get validator
            const validator = getValidator(authData.__typename, 'maxObjectsCheck-delete')
            // Find owner and object type
            const owners = validator.owners(authData);
            // Decrement count for owner
            const ownerId: string | undefined = owners.Organization?.id ?? owners.User?.id;
            if (!ownerId) throw new CustomError(CODE.InternalError, 'Could not find owner ID for object', { code: genErrorCode('0311') });
            counts[authData.__typename] = counts[authData.__typename] || {}
            counts[authData.__typename]![ownerId] = (counts[authData.__typename]![ownerId] || 0) - 1
        }
    }
    // Query the database for the current counts of all objects owned by the user or organization, 
    // and add them to the counts object
    fdsafdsafds
    // Check if any counts exceed the maximum
    for (const objectType of Object.keys(counts)) {
        for (const ownerId of Object.keys(counts[objectType])) {
            fdsafdsafd
        }
    }
}