import { GqlModelType } from "@local/shared";
import { CustomError } from "../../events/error";

/**
 * Resolves any GraphQL union. 
 * This is used by GraphQL to set the __typename field on union objects. 
 * Since a user can query for any fields of an object, this makes it tricky 
 * to determine the type of the object. We get around this by supplementing 
 * every object with our own type field
 */
export const resolveUnion = (object: Record<string, unknown>): `${GqlModelType}` => {
    if (object.__typename) {
        return object.__typename as `${GqlModelType}`;
    } else {
        throw new CustomError("0364", "InternalError", ["en"]);
    }
};
