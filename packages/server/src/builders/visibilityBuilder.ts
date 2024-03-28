import { ModelMap } from "../models/base";
import { combineQueries } from "./combineQueries";
import { VisibilityBuilderProps } from "./types";

/**
 * Assembles visibility query
 */
export const visibilityBuilder = ({
    objectType,
    userData,
    visibility,
}: VisibilityBuilderProps): { [x: string]: any } => {
    // Get validator for object type
    const validator = ModelMap.get(objectType).validate();
    // If visibility is set to public or not defined, 
    // or user is not logged in, or model does not have 
    // the correct data to query for ownership
    if (!visibility || visibility === "Public" || !userData) {
        return validator.visibility.public;
    }
    // If visibility is set to private, query private objects that you own
    else if (visibility === "Private") {
        return combineQueries([validator.visibility.private, validator.visibility.owner(userData.id)]);
    }
    // If visibility is set to own, query all objects that you own
    else if (visibility === "Own") {
        return validator.visibility.owner(userData.id);
    }
    // Otherwise, query all. Be careful with this one, as we don't 
    // want to include private objects that you don't own
    else {
        return combineQueries([{
            OR: [
                validator.visibility.public,
                combineQueries([validator.visibility.private, validator.visibility.owner(userData.id)]),
            ],
        }]);
    }
};
