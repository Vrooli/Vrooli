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
        // throw new CustomError("0680", "InternalError", ["en"], { objectType, funcName });
        if (throwIfNotFound) {
            throw new CustomError("0680", "InternalError", ["en"], { objectType, funcName });
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

export function useVisibility<T extends GenericModelLogic>(
    objectType: GqlModelType | `${GqlModelType}`,
    which: VisibilityType | `${VisibilityType}`,
    data: VisibilityFuncInput,
) {
    const visibilityFunc = getVisibilityFunc<T>(objectType, which);
    return visibilityFunc(data);
}
