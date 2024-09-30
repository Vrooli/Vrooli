import { DUMMY_ID, GqlModelType } from "@local/shared";
import { CustomError } from "../events/error";
import { GenericModelLogic, ModelMap } from "../models/base";
import { combineQueries } from "./combineQueries";
import { VisibilityBuilderProps } from "./types";

function assertFunc<T>(func: T | null, objectType: GqlModelType | `${GqlModelType}`, message: string): asserts func is T {
    if (!func) {
        throw new CustomError("0680", "InternalError", ["en"], { objectType, message });
    }
}

/**
 * Assembles visibility query for Prisma read operations
 */
export function visibilityBuilderPrisma({
    objectType,
    userData,
    visibility,
}: VisibilityBuilderProps): { [x: string]: any } {
    // Get validator for object type
    const validator = ModelMap.get(objectType).validate();
    const publicFunc = validator.visibility.public;
    const privateFunc = validator.visibility.private;
    const ownerFunc = validator.visibility.owner;
    // If visibility is set to public or not defined, 
    // or user is not logged in, or model does not have 
    // the correct data to query for ownership
    if (!visibility || visibility === "Public" || !userData) {
        assertFunc(publicFunc, objectType, "public");
        return publicFunc(DUMMY_ID);
    }
    // If visibility is set to private, query private objects that you own
    else if (visibility === "Private") {
        assertFunc(privateFunc, objectType, "private");
        assertFunc(ownerFunc, objectType, "owner");
        return combineQueries([privateFunc(userData.id), ownerFunc(userData.id)], { mergeMode: "strict" });
    }
    // If visibility is set to own, query all objects that you own
    else if (visibility === "Own") {
        assertFunc(ownerFunc, objectType, "owner");
        return ownerFunc(userData.id);
    }
    // Otherwise, query all. Be careful with this one, as we don't 
    // want to include private objects that you don't own
    else {
        assertFunc(publicFunc, objectType, "public");
        assertFunc(privateFunc, objectType, "private");
        assertFunc(ownerFunc, objectType, "owner");
        return {
            OR: [
                publicFunc(userData.id),
                combineQueries([privateFunc(userData.id), ownerFunc(userData.id)], { mergeMode: "strict" }),
            ],
        };
    }
}

export function useVisibility<T extends GenericModelLogic>(
    objectType: GqlModelType | `${GqlModelType}`,
    which: "public" | "private" | "owner",
    userId: string,
) {
    const validatorFunc = ModelMap.get<T>(objectType).validate().visibility[which];
    assertFunc(validatorFunc, objectType, which);
    return validatorFunc(userId);
}
