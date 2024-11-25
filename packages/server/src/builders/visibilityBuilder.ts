import { DUMMY_ID, GqlModelType, VisibilityType } from "@local/shared";
import { CustomError } from "../events/error";
import { GenericModelLogic, ModelMap } from "../models/base";
import { VisibilityFunc, VisibilityFuncInput } from "../models/types";
import { VisibilityBuilderPrismaResult, VisibilityBuilderProps } from "./types";

// Create a map of visibility types to their corresponding function names
const visibilityFuncNames = {
    [VisibilityType.Own]: "own",
    [VisibilityType.OwnOrPublic]: "ownOrPublic",
    [VisibilityType.OwnPrivate]: "ownPrivate",
    [VisibilityType.OwnPublic]: "ownPublic",
    [VisibilityType.Public]: "public",
} as const;

// Function to get the visibility function based on type
export function getVisibilityFunc<
    ModelLogic extends GenericModelLogic,
    Throw extends boolean = true,
>(
    objectType: GqlModelType | `${GqlModelType}`,
    visibilityType: VisibilityType | `${VisibilityType}`,
    throwIfNotFound: Throw = true as Throw,
): Throw extends true ? VisibilityFunc<any> : (VisibilityFunc<any> | null) {
    const validator = ModelMap.get<ModelLogic>(objectType).validate();
    const funcName = visibilityFuncNames[visibilityType as VisibilityType];
    const visibilityFunc = validator.visibility[funcName];

    if (visibilityFunc === null || visibilityFunc === undefined) {
        if (throwIfNotFound) {
            throw new CustomError("0680", "InternalError", { objectType, funcName });
        } else {
            null;
        }
    }

    return visibilityFunc as Throw extends true ? VisibilityFunc<any> : (VisibilityFunc<any> | null);
}

/**
 * Assembles visibility query for Prisma read operations
 */
export function visibilityBuilderPrisma({
    objectType,
    searchInput,
    userData,
    visibility,
}: VisibilityBuilderProps): VisibilityBuilderPrismaResult {
    const defaultVisibility = (!visibility || !userData) ? VisibilityType.Public : VisibilityType.OwnOrPublic;
    const chosenVisibility = visibility || defaultVisibility;

    const visibilityFunc = getVisibilityFunc(objectType, chosenVisibility);

    const userId = userData?.id || DUMMY_ID;
    const query = visibilityFunc({ searchInput, userId });

    return { query, visibilityUsed: chosenVisibility };
}

export function useVisibility<T extends GenericModelLogic, Throw extends boolean = true>(
    objectType: GqlModelType | `${GqlModelType}`,
    which: VisibilityType | `${VisibilityType}`,
    data: VisibilityFuncInput,
    throwIfNotFound: Throw = true as Throw,
) {
    const visibilityFunc = getVisibilityFunc<T, Throw>(objectType, which, throwIfNotFound);
    if (!visibilityFunc) {
        if (throwIfNotFound) {
            throw new CustomError("0681", "InternalError", { objectType, which });
        } else {
            return null;
        }
    }
    return visibilityFunc(data);
}

/**
 * Loops over an object to generate a list of visibility functions. 
 * Useful for objects that have a lot of relations, such as comments, 
 * resource lists, etc.
 */
export function useVisibilityMapper<ForMapper extends Record<string, string>>(
    which: VisibilityType | `${VisibilityType}`,
    data: VisibilityFuncInput,
    forMapper: ForMapper,
    throwIfNotFound = true,
) {
    return Object.entries(forMapper)
        .map(([key, value]) => { // Find visibility function for each key
            return [value, useVisibility(key as GqlModelType, which, data, throwIfNotFound)];
        })
        .filter(([, visibility]) => visibility) // Remove entries with no visibility
        .map(([key, visibility]) => ({ [key]: visibility })); // Convert to Prisma query format
}
