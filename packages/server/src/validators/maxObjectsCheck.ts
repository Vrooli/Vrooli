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
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { GqlModelType, ObjectLimit, ObjectLimitOwner, ObjectLimitPremium, ObjectLimitPrivacy, SessionUser } from '@shared/consts';
import { PrismaType } from "../types";
import { QueryAction } from "../utils/types";

/**
 * Helper function to check if a count exceeds a number
 */
const checkObjectLimitNumber = (
    count: number,
    limit: number,
    languages: string[]
): void => {
    if (count > limit) {
        throw new CustomError('0352', 'MaxObjectsReached', languages);
    }
}


/**
 * Helper function to check if a count exceeds a premium/noPremium limit
 */
const checkObjectLimitPremium = (
    count: number,
    hasPremium: boolean,
    limit: ObjectLimitPremium,
    languages: string[]
): void => {
    if (hasPremium) checkObjectLimitNumber(count, limit.premium, languages);
    else checkObjectLimitNumber(count, limit.noPremium, languages);
}

/**
 * Helper function to check if a count exceeds a public/private limit
 */
const checkObjectLimitPrivacy = (
    count: number,
    hasPremium: boolean,
    isPrivate: boolean,
    limit: ObjectLimitPrivacy,
    languages: string[]
): void => {
    if (isPrivate) {
        if (typeof limit.private === 'number') checkObjectLimitNumber(count, limit.private, languages);
        else checkObjectLimitPremium(count, hasPremium, limit.private as ObjectLimitPremium, languages);
    }
    else {
        if (typeof limit.public === 'number') checkObjectLimitNumber(count, limit.public, languages);
        else checkObjectLimitPremium(count, hasPremium, limit.public as ObjectLimitPremium, languages);
    }
}

/**
 * Helper function to check if a count exceeds an Organization/User limit
 */
const checkObjectLimitOwner = (
    count: number,
    ownerType: 'User' | 'Organization',
    hasPremium: boolean,
    isPrivate: boolean,
    limit: ObjectLimitOwner,
    languages: string[]
): void => {
    if (ownerType === 'User') {
        if (typeof limit.User === 'number') checkObjectLimitNumber(count, limit.User, languages);
        else if (typeof (limit.User as ObjectLimitPremium).premium !== undefined) checkObjectLimitPremium(count, hasPremium, limit.User as ObjectLimitPremium, languages);
        else checkObjectLimitPrivacy(count, hasPremium, isPrivate, limit.User as ObjectLimitPrivacy, languages);
    }
    else {
        if (typeof limit.Organization === 'number') checkObjectLimitNumber(count, limit.Organization, languages);
        else if (typeof (limit.Organization as ObjectLimitPremium).premium !== undefined) checkObjectLimitPremium(count, hasPremium, limit.Organization as ObjectLimitPremium, languages);
        else checkObjectLimitPrivacy(count, hasPremium, isPrivate, limit.Organization as ObjectLimitPrivacy, languages);
    }
}

/**
 * Helper function to check if a count exceeds the limit
 * @param count The count
 * @param ownerType User or Organization
 * @param hasPremium Whether the user has a premium subscription
 * @param isPrivate Whether to check limit for private or public objects
 * @param limit The limit object. Can be a number, or object with different 
 * @param languages The languages to use for error messages
 * limits depending on premium status, owner type, etc.
 */
const checkObjectLimit = (
    count: number,
    ownerType: 'User' | 'Organization',
    hasPremium: boolean,
    isPrivate: boolean,
    limit: ObjectLimit,
    languages: string[],
): void => {
    if (typeof limit === 'number') checkObjectLimitNumber(count, limit, languages);
    else if (typeof (limit as ObjectLimitPremium).premium !== undefined) checkObjectLimitPremium(count, hasPremium, limit as ObjectLimitPremium, languages);
    else if (typeof (limit as ObjectLimitPrivacy).private !== undefined) checkObjectLimitPrivacy(count, hasPremium, isPrivate, limit as ObjectLimitPrivacy, languages);
    else checkObjectLimitOwner(count, ownerType, hasPremium, isPrivate, limit as ObjectLimitOwner, languages);
}

// TODO Would be nice if we could check max number of relations for an object, not just the absolute number of the object. 
// For example, we could limit the versions on a root object
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
    authDataById: { [id: string]: { __typename: `${GqlModelType}`, [x: string]: any } },
    idsByAction: { [key in QueryAction]?: string[] },
    prisma: PrismaType,
    userData: SessionUser,
) {
    // Initialize counts. This is used to count how many objects a user or organization will have after every action is applied.
    const counts: { [key in GqlModelType]?: { [ownerId: string]: { private: number, public: number } } } = {}
    // Loop through every "Create" action, and increment the count for the object type
    if (idsByAction.Create) {
        for (const id of idsByAction.Create) {
            // Get auth data
            const authData = authDataById[id]
            // Get validator
            const { validate } = getLogic(['validate'], authData.__typename, userData.languages, 'maxObjectsCheck-create')
            // Find owner and object type
            const owners = validate.owner(authData);
            // Increment count for owner
            const ownerId: string | undefined = owners.Organization?.id ?? owners.User?.id;
            if (!ownerId) throw new CustomError('0310', 'InternalError', userData.languages);
            // Initialize shape of counts for this owner
            counts[authData.__typename] = counts[authData.__typename] || {}
            counts[authData.__typename]![ownerId] = counts[authData.__typename]![ownerId] || { private: 0, public: 0 }
            // Determine if object is public
            const isPublic = validate.isPublic(authData, userData.languages);
            // Increment count
            counts[authData.__typename]![ownerId][isPublic ? 'public' : 'private']++;
        }
    }
    // Ignore count for updates, as that doesn't change the total number of objects
    // Loop through every "Delete" action, and decrement the count for the object type
    if (idsByAction.Delete) {
        for (const id of idsByAction.Delete) {
            // Get auth data
            const authData = authDataById[id]
            // Get validator
            const { validate } = getLogic(['validate'], authData.__typename, userData.languages, 'maxObjectsCheck-delete')
            // Find owner and object type
            const owners = validate.owner(authData);
            // Decrement count for owner
            const ownerId: string | undefined = owners.Organization?.id ?? owners.User?.id;
            if (!ownerId) throw new CustomError('0311', 'InternalError', userData.languages);
            // Initialize shape of counts for this owner
            counts[authData.__typename] = counts[authData.__typename] || {}
            counts[authData.__typename]![ownerId] = counts[authData.__typename]![ownerId] || { private: 0, public: 0 }
            // Determine if object is public
            const isPublic = validate.isPublic(authData, userData.languages);
            // Decrement count
            counts[authData.__typename]![ownerId][isPublic ? 'public' : 'private']--;
        }
    }
    // Add counts for all existing objects, then check if any limits will be exceeded
    // Loop through every object type in the counts object
    for (const objectType of Object.keys(counts)) {
        // Get delegate and validate functions for the object type
        const { delegate, validate } = getLogic(['delegate', 'validate'], objectType as GqlModelType, userData.languages, 'maxObjectsCheck-existing');
        // Loop through every owner in the counts object
        for (const ownerId in counts[objectType]!) {
            // Query the database for the current counts of objects owned by the owner
            let currCountPrivate = await delegate(prisma).count({ where: validate.visibility.private });
            let currCountPublic = await delegate(prisma).count({ where: validate.visibility.public });
            // Add count obtained from add and deletes to the current counts
            currCountPrivate += counts[objectType]![ownerId].private;
            currCountPublic += counts[objectType]![ownerId].public;
            // Now that we have the total counts for both private and public objects, check if either exceeds the maximum
            const maxObjects = validate.maxObjects;
            const ownerType = userData.id === ownerId ? 'User' : 'Organization';
            const hasPremium = userData.hasPremium;
            checkObjectLimit(currCountPrivate, ownerType, hasPremium, true, maxObjects, userData.languages);
            checkObjectLimit(currCountPublic, ownerType, hasPremium, false, maxObjects, userData.languages);
        }
    }
}