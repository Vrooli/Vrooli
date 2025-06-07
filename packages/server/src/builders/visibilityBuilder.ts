import { ApiKeyPermission, DUMMY_ID, type ModelType, VisibilityType } from "@vrooli/shared";
import { RequestService } from "../auth/request.js";
import { CustomError } from "../events/error.js";
import { type GenericModelLogic, ModelMap } from "../models/base/index.js";
import { type VisibilityFunc, type VisibilityFuncInput } from "../models/types.js";
import { type VisibilityBuilderPrismaResult, type VisibilityBuilderProps } from "./types.js";

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
    objectType: ModelType | `${ModelType}`,
    visibilityType: VisibilityType | `${VisibilityType}`,
    throwIfNotFound: Throw = true as Throw,
): Throw extends true ? VisibilityFunc<any> : (VisibilityFunc<any> | null) {
    const validator = ModelMap.get<ModelLogic>(objectType).validate();
    // Ensure visibilityType is a valid key before accessing visibilityFuncNames
    const funcName = (visibilityType in visibilityFuncNames) ? visibilityFuncNames[visibilityType as VisibilityType] : undefined;

    if (!funcName) {
        if (throwIfNotFound) {
            // Use "InternalError" key, pass details in meta
            throw new CustomError("0680", "InternalError", { message: "Invalid visibility type requested.", objectType, visibilityType });
        } else {
            // Use type assertion 'as any' for conciseness
            return null as any;
        }
    }

    let visibilityFunc = validator.visibility[funcName];

    // Try fallbacks
    if (!visibilityFunc && visibilityType === VisibilityType.OwnOrPublic) {
        visibilityFunc = validator.visibility.own || validator.visibility.public;
    }

    if (visibilityFunc === null || visibilityFunc === undefined) {
        // Check if the function is explicitly null (not supported) or just missing
        const isUnsupported = funcName in validator.visibility && validator.visibility[funcName] === null;
        const message = `Visibility function '${funcName}' ${isUnsupported ? "is not supported" : "not found"} for ${objectType}.`;
        if (throwIfNotFound) {
            // Use "InternalError" key, pass details in meta
            throw new CustomError("0780", "InternalError", { message, objectType, funcName });
        } else {
            // Use type assertion 'as any' for conciseness
            return null as any;
        }
    }

    return visibilityFunc;
}

// Defines the restrictiveness hierarchy (lower number = more restrictive)
const visibilityOrder: Record<VisibilityType, number> = {
    [VisibilityType.Public]: 1,
    // Note: OwnPublic/OwnPrivate might need adjustment depending on exact semantics
    [VisibilityType.OwnPublic]: 2,
    [VisibilityType.OwnPrivate]: 3,
    [VisibilityType.Own]: 4,
    [VisibilityType.OwnOrPublic]: 5,
};

/**
 * Determines the most restrictive visibility level allowed.
 */
function getEffectiveVisibility(
    requestedVisibility: VisibilityType | null | undefined,
    maxAllowedVisibility: VisibilityType,
): VisibilityType {
    if (!requestedVisibility) {
        return maxAllowedVisibility; // Default to the max allowed if nothing specific is requested
    }

    // Ensure both visibilities are valid before comparing
    if (!(requestedVisibility in visibilityOrder) || !(maxAllowedVisibility in visibilityOrder)) {
        // Handle invalid visibility types if necessary, maybe default to Public or throw
        console.error("Invalid visibility type encountered in getEffectiveVisibility", { requestedVisibility, maxAllowedVisibility });
        return VisibilityType.Public; // Safe fallback
    }

    // If the requested level is less or equally restrictive than the max allowed, use the requested level.
    // Otherwise, clamp down to the maximum allowed level.
    if (visibilityOrder[requestedVisibility] <= visibilityOrder[maxAllowedVisibility]) {
        return requestedVisibility;
    } else {
        return maxAllowedVisibility;
    }
}

/**
 * Assembles visibility query for Prisma read operations, respecting permissions.
 */
export function visibilityBuilderPrisma({
    objectType,
    req,
    searchInput,
    visibility, // The caller's requested visibility level (optional)
}: VisibilityBuilderProps): VisibilityBuilderPrismaResult {
    const { permissions, userData } = RequestService.getRequestPermissions(req);

    // 1. Determine the *maximum* visibility allowed based on permissions
    let maxAllowedVisibility: VisibilityType;
    const isApiKey = Object.keys(permissions).length > 0; // Check if API key permissions exist

    if (isApiKey) {
        // API Key authentication
        if (permissions[ApiKeyPermission.ReadPrivate]) {
            // ReadPrivate likely implies ability to see own + public things
            maxAllowedVisibility = VisibilityType.OwnOrPublic;
        } else if (permissions[ApiKeyPermission.ReadPublic]) {
            maxAllowedVisibility = VisibilityType.Public;
        } else {
            // API key exists but grants neither known read permission. Default to most restrictive.
            maxAllowedVisibility = VisibilityType.Public;
        }
    } else if (userData) {
        // Logged-in user session (no API key)
        maxAllowedVisibility = VisibilityType.OwnOrPublic;
    } else {
        // Logged-out user
        maxAllowedVisibility = VisibilityType.Public;
    }

    // 2. Determine the *effective* visibility by taking the more restrictive
    //    of the requested visibility and the max allowed visibility.
    const chosenVisibility = getEffectiveVisibility(visibility, maxAllowedVisibility);

    // 3. Get the corresponding visibility function, falling back if necessary
    let visibilityFunc = getVisibilityFunc(objectType, chosenVisibility, false); // Don't throw immediately
    let finalVisibilityUsed = chosenVisibility; // Track the final visibility level used

    // If the chosen function doesn't exist, try falling back to Public (unless already Public)
    if (!visibilityFunc && chosenVisibility !== VisibilityType.Public) {
        // Use optional chaining for safe access, although it should exist if chosenVisibility is valid
        const funcName = visibilityFuncNames[chosenVisibility];
        console.warn(`Visibility function '${funcName || chosenVisibility}' not found for ${objectType}. Falling back to Public.`);
        const publicFunc = getVisibilityFunc(objectType, VisibilityType.Public, false);
        if (publicFunc) {
            visibilityFunc = publicFunc;
            finalVisibilityUsed = VisibilityType.Public; // Update the final visibility used
        } else {
            // Use "InternalError" key, pass details in meta
            const message = `Neither requested visibility ('${chosenVisibility}') nor fallback 'Public' function found for ${objectType}.`;
            throw new CustomError("0781", "InternalError", { message, objectType, chosenVisibility });
        }
    } else if (!visibilityFunc && chosenVisibility === VisibilityType.Public) {
        // Use "InternalError" key, pass details in meta
        const message = `Required 'Public' visibility function not found for ${objectType}.`;
        throw new CustomError("0782", "InternalError", { message, objectType });
    }

    // Ensure we actually got a function before proceeding
    if (!visibilityFunc) {
        // This case should ideally be caught by the checks above, but safeguards are good.
        const message = `Failed to obtain a valid visibility function for ${objectType}.`;
        throw new CustomError("0783", "InternalError", { message, objectType, chosenVisibility });
    }


    // 4. Execute the function
    const userId = userData?.id || DUMMY_ID; // Pass userId for context, even for Public visibility
    const query = visibilityFunc({ searchInput, userId });

    return { query, visibilityUsed: finalVisibilityUsed };
}

export function useVisibility<T extends GenericModelLogic, Throw extends boolean = true>(
    objectType: ModelType | `${ModelType}`,
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
 * Useful for objects that have a lot of relations, such as comments
 */
export function useVisibilityMapper<ForMapper extends Record<string, string>>(
    which: VisibilityType | `${VisibilityType}`,
    data: VisibilityFuncInput,
    forMapper: ForMapper,
    throwIfNotFound = true,
) {
    return Object.entries(forMapper)
        .map(([key, value]) => { // Find visibility function for each key
            return [value, useVisibility(key as ModelType, which, data, throwIfNotFound)];
        })
        .filter(([, visibility]) => visibility) // Remove entries with no visibility
        .map(([key, visibility]) => ({ [key]: visibility })); // Convert to Prisma query format
}
