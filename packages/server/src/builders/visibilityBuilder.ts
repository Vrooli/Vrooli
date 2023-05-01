import { getLogic } from "../getters";
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
    const { validate } = getLogic(["validate"], objectType, userData?.languages ?? ["en"], "visibilityBuilder");
    // If visibility is set to public or not defined, 
    // or user is not logged in, or model does not have 
    // the correct data to query for ownership
    if (!visibility || visibility === "Public" || !userData) {
        return validate.visibility.public;
    }
    // If visibility is set to private, query private objects that you own
    else if (visibility === "Private") {
        return combineQueries([validate.visibility.private, validate.visibility.owner(userData.id)]);
    }
    // If visibility is set to own, query all objects that you own
    else if (visibility === "Own") {
        return validate.visibility.owner(userData.id);
    }
    // Otherwise, query all. Be careful with this one, as we don't 
    // want to include private objects that you don't own
    else {
        return combineQueries([{
            OR: [
                validate.visibility.public,
                combineQueries([validate.visibility.private, validate.visibility.owner(userData.id)]),
            ],
        }]);
    }
}
