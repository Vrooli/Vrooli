import { getValidator } from "../getters";
import { combineQueries } from "./combineQueries";
import { VisibilityBuilderProps } from "./types";

/**
 * Assembles visibility query
 */
export function visibilityBuilder({
    objectType,
    userData,
    visibility,
}: VisibilityBuilderProps): { [x: string]: any } {
    // Get validator for object type
    const validator = getValidator(objectType, userData?.languages ?? ['en'], 'visibilityBuilder');
    // If visibility is set to public or not defined, 
    // or user is not logged in, or model does not have 
    // the correct data to query for ownership
    if (!visibility || visibility === 'Public' || !userData) {
        return { isPrivate: false };
    }
    // If visibility is set to private, query private objects that you own
    else if (visibility === 'Private') {
        return combineQueries([
            { isPrivate: true }, 
            validator.ownerOrMemberWhere(userData.id)
        ])
    }
    // Otherwise, must be set to All
    else {
        let query = validator.ownerOrMemberWhere(userData.id);
        // If query has OR field with an array value, add isPrivate: false to OR array
        if ('OR' in query && Array.isArray(query.OR)) {
            query.OR.push({ isPrivate: false });
        }
        // Otherwise, wrap query in OR with isPrivate: false
        else {
            query = { OR: [query, { isPrivate: false }] };
        }
        return query;
    }
}