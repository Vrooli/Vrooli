import { getLogic } from "../getters";
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
    const { validate } = getLogic(['validate'], objectType, userData?.languages ?? ['en'], 'visibilityBuilder');
    // If visibility is set to public or not defined, 
    // or user is not logged in, or model does not have 
    // the correct data to query for ownership
    if (!visibility || visibility === 'Public' || !userData) {
        return validate.visibility.public;
    }
    // If visibility is set to private, query private objects that you own
    else if (visibility === 'Private') {
        return validate.visibility.private
    }
    // Otherwise, must be set to all
    else {
        return validate.visibility.owner(userData.id);
    }
}