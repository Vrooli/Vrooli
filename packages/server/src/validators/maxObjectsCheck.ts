/**
 * This file handles checking if a cud action violates any max object limits. 
 * Limits are defined for both users and teams, and can be raised 
 * by the standard premium subscription, or a custom subscription.
 * 
 * The general idea is that users can have a limited number of private objects, and
 * a larger number of public objects. Teams can have even more private objects, and 
 * even more public objects. The limits should be just low enough to encourage people to make public 
 * objects and transfer them to teams, but not so low that it's impossible to use the platform.
 * 
 * We want objects to be owned by teams rather than users, as this means the objects are tied to 
 * the team's governance structure.
 */
import { GqlModelType, ObjectLimit, ObjectLimitOwner, ObjectLimitPremium, ObjectLimitPrivacy } from "@local/shared";
import { PrismaDelegate } from "../builders/types";
import { getVisibilityFunc } from "../builders/visibilityBuilder";
import { prismaInstance } from "../db/instance";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { SessionUserToken } from "../types";
import { authDataWithInput } from "../utils/authDataWithInput";
import { AuthDataById } from "../utils/getAuthenticatedData";
import { getParentInfo } from "../utils/getParentInfo";
import { InputsById, QueryAction } from "../utils/types";

/**
 * Helper function to check if a count exceeds a number
 */
function checkObjectLimitNumber(
    count: number,
    limit: number,
    languages: string[],
): void {
    if (count > limit) {
        throw new CustomError("0352", "MaxObjectsReached", languages);
    }
}


/**
 * Helper function to check if a count exceeds a premium/noPremium limit
 */
function checkObjectLimitPremium(
    count: number,
    hasPremium: boolean,
    limit: ObjectLimitPremium,
    languages: string[],
): void {
    if (hasPremium) checkObjectLimitNumber(count, limit.premium, languages);
    else checkObjectLimitNumber(count, limit.noPremium, languages);
}

/**
 * Helper function to check if a count exceeds a public/private limit
 */
function checkObjectLimitPrivacy(
    count: number,
    hasPremium: boolean,
    isPublic: boolean,
    limit: ObjectLimitPrivacy,
    languages: string[],
): void {
    if (isPublic) {
        if (typeof limit.public === "number") checkObjectLimitNumber(count, limit.public, languages);
        else checkObjectLimitPremium(count, hasPremium, limit.public as ObjectLimitPremium, languages);
    }
    else {
        if (typeof limit.private === "number") checkObjectLimitNumber(count, limit.private, languages);
        else checkObjectLimitPremium(count, hasPremium, limit.private as ObjectLimitPremium, languages);
    }
}

/**
 * Helper function to check if a count exceeds an Team/User limit
 */
function checkObjectLimitOwner(
    count: number,
    ownerType: "User" | "Team",
    hasPremium: boolean,
    isPublic: boolean,
    limit: ObjectLimitOwner,
    languages: string[],
): void {
    if (ownerType === "User") {
        if (typeof limit.User === "number") checkObjectLimitNumber(count, limit.User, languages);
        else if (typeof (limit.User as ObjectLimitPremium).premium !== undefined) checkObjectLimitPremium(count, hasPremium, limit.User as ObjectLimitPremium, languages);
        else checkObjectLimitPrivacy(count, hasPremium, isPublic, limit.User as ObjectLimitPrivacy, languages);
    }
    else {
        if (typeof limit.Team === "number") checkObjectLimitNumber(count, limit.Team, languages);
        else if (typeof (limit.Team as ObjectLimitPremium).premium !== undefined) checkObjectLimitPremium(count, hasPremium, limit.Team as ObjectLimitPremium, languages);
        else checkObjectLimitPrivacy(count, hasPremium, isPublic, limit.Team as ObjectLimitPrivacy, languages);
    }
}

/**
 * Helper function to check if a count exceeds the limit
 */
function checkObjectLimit({
    count,
    ownerType,
    hasPremium,
    isPublic,
    limit,
    languages,
}: {
    /** The Count */
    count: number,
    /** The owner type (i.e. "User" or "Team") */
    ownerType: "User" | "Team",
    /** Whether the user has a premium subscription */
    hasPremium: boolean,
    /** Whether the object is public */
    isPublic: boolean,
    /** The languages to use for error messages */
    languages: string[],
    /** The limit object. Can be a number, or object with different limits depending on premium status, owner type, etc. */
    limit: ObjectLimit,
}): void {
    if (typeof limit === "number") checkObjectLimitNumber(count, limit, languages);
    else if (typeof (limit as ObjectLimitPremium).premium !== undefined) checkObjectLimitPremium(count, hasPremium, limit as ObjectLimitPremium, languages);
    else if (typeof (limit as ObjectLimitPrivacy).private !== undefined) checkObjectLimitPrivacy(count, hasPremium, isPublic, limit as ObjectLimitPrivacy, languages);
    else checkObjectLimitOwner(count, ownerType, hasPremium, isPublic, limit as ObjectLimitOwner, languages);
}

// TODO Would be nice if we could check max number of relations for an object, not just the absolute number of the object. 
// For example, we could limit the versions on a root object
/**
 * Validates that the user will not exceed any maximum object limits after the given action. Checks both 
 * for personal limits and team limits, factoring in the user or team's premium status. 
 * Throws an error if a limit will be exceeded.
 * @param authDataById Map of all queried data required to validate permissions, keyed by ID.
 * @param idsByAction Map of object IDs to validate permissions for, keyed by action. We store actions this way (instead of keyed by ID) 
 * in case one ID is used for multiple actions.
 * @param userId ID of user requesting permissions
 */
export async function maxObjectsCheck(
    inputsById: InputsById,
    authDataById: AuthDataById,
    idsByAction: { [key in QueryAction]?: string[] },
    userData: SessionUserToken,
) {
    // Initialize counts. This is used to count how many objects a user or team will have after every action is applied.
    const counts: { [key in GqlModelType]?: { [ownerId: string]: { private: number, public: number } } } = {};
    // Loop through every "Create" action, and increment the count for the object type
    if (idsByAction.Create) {
        for (const id of idsByAction.Create) {
            const typename = inputsById[id].node.__typename;
            // Get validator
            const validator = ModelMap.get(typename, true, "maxObjectsCheck 2").validate();
            // Creates shouldn't have authData, so we'll only use the input data. 
            // We still need to pass this through authDataWithInput to convert relationships to the correct shape
            const combinedData = authDataWithInput(inputsById[id].input as object, {}, inputsById, authDataById);
            // Find owner and object type
            const owners = validator.owner(combinedData, userData.id);
            // Increment count for owner. We can assume we're the owner if no owner was provided
            const ownerId: string | undefined = owners.Team?.id ?? owners.User?.id ?? userData.id;
            // Initialize shape of counts for this owner
            counts[typename] = counts[typename] || {};
            counts[typename]![ownerId] = counts[typename]![ownerId] || { private: 0, public: 0 };
            // Determine if object is public
            const isPublic = validator.isPublic(combinedData, (...rest) => getParentInfo(...rest, inputsById), userData.languages);
            // Increment count
            counts[typename]![ownerId][isPublic ? "public" : "private"]++;
        }
    }
    // Ignore count for updates, as that doesn't change the total number of objects
    // Loop through every "Delete" action, and decrement the count for the object type
    if (idsByAction.Delete) {
        for (const id of idsByAction.Delete) {
            // Get auth data
            const authData = authDataById[id];
            // Get validator
            const validator = ModelMap.get(authData.__typename, true, "maxObjectsCheck 2").validate();
            // Find owner and object type
            const owners = validator.owner(authData, userData.id);
            // Decrement count for owner
            const ownerId: string | undefined = owners.Team?.id ?? owners.User?.id;
            if (!ownerId) throw new CustomError("0311", "InternalError", userData.languages);
            // Initialize shape of counts for this owner
            counts[authData.__typename] = counts[authData.__typename] || {};
            counts[authData.__typename]![ownerId] = counts[authData.__typename]![ownerId] || { private: 0, public: 0 };
            // Determine if object is public
            const isPublic = validator.isPublic(authData, (...rest) => getParentInfo(...rest, inputsById), userData.languages);
            // Decrement count
            counts[authData.__typename]![ownerId][isPublic ? "public" : "private"]--;
        }
    }
    // Add counts for all existing objects, then check if any limits will be exceeded
    // Loop through every object type in the counts object
    for (const objectType of Object.keys(counts)) {
        // Get delegate and validate functions for the object type
        const delegator = prismaInstance[ModelMap.get(objectType as GqlModelType, true, "maxObjectsCheck 3").dbTable] as PrismaDelegate;
        const validator = ModelMap.get(objectType as GqlModelType, true, "maxObjectsCheck 4").validate();
        // Loop through every owner in the counts object
        for (const ownerId in counts[objectType]!) {
            // Query the database for the current counts of objects owned by the owner
            const searchData = { searchInput: {}, userId: ownerId };
            let currCountPrivate = 0;
            let currCountPublic = 0;
            // Some objects don't support private/public, so may have null visibility functions. We can ignore these.
            const privateVisibility = getVisibilityFunc(objectType as GqlModelType, "OwnPrivate", false);
            const publicVisibility = getVisibilityFunc(objectType as GqlModelType, "OwnPublic", false);
            if (privateVisibility) currCountPrivate = await delegator.count({ where: privateVisibility(searchData) });
            if (publicVisibility) currCountPublic = await delegator.count({ where: publicVisibility(searchData) });
            // Add count obtained from add and deletes to the current counts
            currCountPrivate += counts[objectType]![ownerId].private;
            currCountPublic += counts[objectType]![ownerId].public;
            // Now that we have the total counts for both private and public objects, check if either exceeds the maximum
            const maxObjects = validator.maxObjects;
            const ownerType = userData.id === ownerId ? "User" : "Team";
            const hasPremium = userData.hasPremium;
            checkObjectLimit({
                count: currCountPrivate,
                hasPremium,
                isPublic: false,
                languages: userData.languages,
                limit: maxObjects,
                ownerType,
            });
            checkObjectLimit({
                count: currCountPublic,
                hasPremium,
                isPublic: true,
                languages: userData.languages,
                limit: maxObjects,
                ownerType,
            });
        }
    }
}
