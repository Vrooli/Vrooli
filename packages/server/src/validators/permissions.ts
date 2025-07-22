// AI_CHECK: TYPE_SAFETY=security-critical-1 | LAST: 2025-07-04 - Fixed defaultPermissions return type to include required QueryAction methods and added RawOwnerData type for bigint IDs
import { DUMMY_ID, type ModelType, type SessionUser } from "@vrooli/shared";
import { permissionsSelectHelper } from "../builders/permissionsSelectHelper.js";
import { type PrismaDelegate } from "../builders/types.js";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { ModelMap } from "../models/base/index.js";
// AI_CHECK: TYPE_SAFETY=phase2-permissions | LAST: 2025-07-04 - Fixed type safety issues with ModelMap access
import { type AuthDataById, type AuthDataItem } from "../utils/getAuthenticatedData.js";
import { hasTypename } from "../utils/typeGuards.js";
import { type InputsById, type QueryAction } from "../utils/types.js";

/**
 * Handles setting and interpreting permission policies for teams and their objects. Permissions are stored as stringified JSON 
 * in the database (in case we change the structure later), which means we must query for permissions ahead of a crud operation, and then validate the
 * permissions against the operation.
 * 
 * NOTE: When an object is owned by a user instead of a team, there is no permissioning required. The user is the owner, and can do whatever they want.
 * 
 * Objects which have permissions are: 
 * - Team
 * - Member (of a team)
 * - Role (of a team)
 * - Project
 * - Routine
 * - Code
 * - Standard
 * 
 * Multiple permissions must be checked, depending on the object type and its relationship to the user. Permissions can also override each other. 
 * Here is every object type and its permissions hierarchy (from most important to least important):
 * - Team -> Team
 * - Member -> Team
 * - Role -> Team
 * - Project -> Member, Role, Project, Team
 * - Routine -> Member, Role, Routine, Team
 * - Code -> Member, Role, Code, Team
 * - Standard -> Member, Role, Standard, Team
 * 
 * Example: You try to update a project which belongs to a team. First we check if the team permits this (is not deleted, not locked, etc). 
 * Then we check if you have a role in the team, and if that role allows you to update projects. Then we check if you have a membership in the team 
 * and if that membership explicitly excludes you from updating projects. Then we check if the project itself allows you to update it (is not deleted, not locked, etc). 
 * It's a lot of checks, but it's necessary to ensure that permissions are enforced correctly.
 */
export type PermissionType = "Create" | "Read" | "Update" | "Delete" | "Fork" | "Report" | "Run";

/**
 * Permissions policy fields which are common to all policies in a team. All related to how and how often the policy can be updated.
 */
export type PolicyPart = {
    /**
     * Allows you to prevent this policy from being updated, without a time limit.
     */
    isLocked?: boolean;
    /**
    * Allows you to limit how frequently this policy can be updated.
    */
    lockedUntil?: Date;
    /**
     * Specifies a routine used to update this policy. Since routines can provide context 
     * and voting mechanisms, attaching a routine to a policy allows you to update it in a
     * more controlled way.
     */
    routineVersionId?: string;
    /**
     * Specifies timeout length after which this policy can be updated again. Used in conjunction with lockedUntil.
     */
    timeout?: number;
};

/**
 * Permissions policy fields which are common to all nested policies in a team.
 */
export type NestedPolicyPart = PolicyPart & {
    /**
     * Custom maximum number allowed. Still have to obey maximum specified globally, which is determined by your premium status
     */
    maxAmount?: number;
    whoCanAdd?: MemberRolePolicy;
    whoCanDelete?: MemberRolePolicy;
    whoCanUpdate?: MemberRolePolicy;
    whoCanChangePolicy?: MemberRolePolicy;
};

/**
 * Permissions policy for a specific code of a team. Useful to lock down a code's permissions
 */
export type CodePolicy = {
    version: number;
    whoCanAdd?: MemberRolePolicy;
    whoCanDelete?: MemberRolePolicy;
    whoCanUpdate?: MemberRolePolicy;
};

/**
 * Permissions policy fields to specify which members/roles are allowed to perform a certain action. 
 * A user only needs to satisfy either member permissions or role permissions to perform the action.
 */
export type MemberRolePolicy = {
    /**
     * Member IDs which are allowed to add data referenced by policy. If not specified or empty, all members are allowed 
     * except those in the deny list. If roles specified, refer to role permissions.
     */
    membersAllow?: string[];
    membersDeny?: string[];
    /**
    * Role IDs which are allowed to add. Same logic as members.
    */
    rolesAllow?: string[];
    rolesDeny?: string[];
};

/**
 * Permissions policy for a specific member of a team. Useful to lock down a member's permissions, or to
 * give them special permissions.
 */
export type MemberPolicy = {
    version: number;
    /**
     * Gives admin privileges. Admins can do anything, including change the team's policy.
     */
    isAdmin?: boolean;
    isLocked?: boolean;
    lockedUntil?: Date;
}

/**
 * Permissions policy for a specific role of a team. Useful to lock down a role's permissions, or to
 * give them special permissions.
 */
export type RolePolicy = {
    version: number;
    /**
     * Gives admin privileges. Admins can do anything, including change the team's policy.
     */
    isAdmin?: boolean;
    isLocked?: boolean;
    lockedUntil?: Date;
}

/**
 * Permissions policy for a specific project of a team. Useful to lock down a project's permissions
 */
export type ProjectPolicy = {
    version: number;
    whoCanAdd?: MemberRolePolicy;
    whoCanDelete?: MemberRolePolicy;
    whoCanUpdate?: MemberRolePolicy;
}

/**
 * Permissions policy for a specific routine of a team. Useful to lock down a routine's permissions
 */
export type RoutinePolicy = {
    version: number;
    whoCanAdd?: MemberRolePolicy;
    whoCanDelete?: MemberRolePolicy;
    whoCanUpdate?: MemberRolePolicy;
}

/**
 * Permissions policy for a specific standard of a team. Useful to lock down a standard's permissions
 */
export type StandardPolicy = {
    version: number;
    whoCanAdd?: MemberRolePolicy;
    whoCanDelete?: MemberRolePolicy;
    whoCanUpdate?: MemberRolePolicy;
};

/**
 * Permissions policy for entire team. Team itself can have same fields in TeamPolicyPart, but can also 
 * specify more specific permissions that apply to parts of the team.
 */
export type TeamPolicy = PolicyPart & {
    /**
     * Policy schema version. Used to help us read JSON properly and migrate policy shapes in the future, if necessary.
     */
    version: number;
    members?: NestedPolicyPart & {
        /**
         * One or more routines which a user must complete before they can be added to the team.
         */
        onboardingRoutineVersionIds?: string[];
        /**
         * One or more projects which a user must complete before they can be added to the team.
         */
        onboardingProjectIds?: string[];
    };
    roles?: NestedPolicyPart;
    info?: NestedPolicyPart;
    codes?: NestedPolicyPart;
    issues?: NestedPolicyPart;
    notes?: NestedPolicyPart;
    projects?: NestedPolicyPart;
    routines?: NestedPolicyPart;
    standards?: NestedPolicyPart;
    voting?: NestedPolicyPart & {
        /**
         * Whether to factor in a user's reputation when they vote.
         */
        factorInReputation?: boolean;
        /**
         * Whether to factor in how many times a user has voted on this object when they vote. 
         * The more votes, the more weight their vote has (up to a certain point). This acts 
         * as a way to measure experience.
         */
        factorInVotingExperience?: boolean;
    };
    data?: NestedPolicyPart & {
        // TODO add policies to specify how data is stored and accessed
    };
    treasury?: NestedPolicyPart & {
        // TODO add policies to specify how treasury is managed
    };
};

type ResolvedPermissions = { [x in Exclude<QueryAction, "Create"> as `can${x}`]: boolean } & { canReact: boolean };

/**
* Helper function to find the parent validation data for an object
* @param id ID of object to find parent for
* @param typename Only return parent if it matches this typename
* @param inputsById Map of all input data, keyed by ID
* @returns Input data for parent object, or undefined if parent doesn't exist or doesn't match typename
*/
export function getParentInfo(id: string, typename: `${ModelType}`, inputsById: InputsById): Record<string, unknown> | undefined {
    const node = inputsById[id]?.node;
    if (node?.__typename !== typename) return undefined;
    if (!node?.parent) return undefined;
    const parentInput = inputsById[node.parent.id]?.input;
    return (parentInput && typeof parentInput === "object") ? parentInput as Record<string, unknown> : undefined;
}

/**
 * Type for raw owner data from database (with bigint IDs)
 */
export interface RawOwnerData {
    Team?: {
        id: bigint;
        members?: Array<{
            userId: bigint;
            isAdmin: boolean;
        }>;
    } | null;
    User?: {
        id: bigint;
    } | null;
}

/**
 * Type for owner object structure (with string IDs)
 */
export interface OwnerData {
    Team?: {
        id: string;
        members?: Array<{
            userId: string;
            isAdmin: boolean;
        }>;
    } | null;
    User?: {
        id: string;
    } | null;
}

/**
 * Checks if the user has admin privileges on an object's creator/owner
 */
export function isOwnerAdminCheck(
    owner: OwnerData,
    userId: string | null | undefined,
): boolean {
    // Can't be an admin if not logged in
    if (userId === null || userId === undefined) return false;
    // If the owner is a user, check id
    if (owner.User) return owner.User.id.toString() === userId;
    // If the owner is a team, check if you're a member with "isAdmin" set to true
    if (owner.Team) {
        return owner.Team.members ?
            owner.Team.members.some((member) => member.userId.toString() === userId && member.isAdmin) :
            false;
    }
    // If the owner is neither a user nor a team, return false
    return false;
}

/**
 * Holds common permission resolvers, which can apply to most models. 
 * Some models may have custom resolvers, which can easily be set in 
 * their ModelLogic object.
 */
// AI_CHECK: TYPE_SAFETY=server-validators-type-safety-maintenance-2 | LAST: 2025-07-04 - Fixed return type to match permissionResolvers requirements
type DefaultPermissionsReturn = {
    canConnect: () => boolean;
    canDelete: () => boolean;
    canDisconnect: () => boolean;
    canRead: () => boolean;
    canUpdate: () => boolean;
} & Record<string, () => boolean>;

export function defaultPermissions({
    isAdmin,
    isDeleted,
    isLoggedIn,
    isPublic,
}: { isAdmin: boolean, isDeleted: boolean, isLoggedIn: boolean, isPublic: boolean }): DefaultPermissionsReturn {
    return {
        // Required QueryAction permissions
        canConnect: () => isLoggedIn && !isDeleted && (isAdmin || isPublic),
        canDelete: () => isLoggedIn && !isDeleted && isAdmin,
        canDisconnect: () => isLoggedIn,
        canRead: () => !isDeleted && (isPublic || isAdmin),
        canUpdate: () => isLoggedIn && !isDeleted && isAdmin,
        // Additional common permissions
        canBookmark: () => isLoggedIn && !isDeleted && (isAdmin || isPublic),
        canComment: () => isLoggedIn && !isDeleted && (isAdmin || isPublic),
        canCopy: () => isLoggedIn && !isDeleted && (isAdmin || isPublic),
        canReport: () => isLoggedIn && !isAdmin && !isDeleted && isPublic,
        canRun: () => !isDeleted && (isAdmin || isPublic),
        canShare: () => !isDeleted && (isAdmin || isPublic),
        canTransfer: () => isLoggedIn && isAdmin && !isDeleted,
        canUse: () => isLoggedIn && !isDeleted && (isAdmin || isPublic),
        canReact: () => isLoggedIn && !isDeleted && (isAdmin || isPublic),
    };
}

/**
 * Filters defaultPermissions to only include the permissions that a specific model needs.
 * This ensures type safety by only returning the permission fields that exist in the model's "You" type.
 * @param permissions The full permissions object from defaultPermissions
 * @param allowedKeys Array of permission keys that the model supports
 * @returns Filtered permissions object with only the specified keys
 */
export function filterPermissions<T extends Record<string, () => any>>(
    permissions: DefaultPermissionsReturn,
    allowedKeys: readonly string[],
): T {
    const filtered: any = {};
    for (const key of allowedKeys) {
        if (key in permissions) {
            filtered[key] = permissions[key];
        }
    }
    return filtered;
}

/**
 * Calculates and returns permissions for a given object based on user and auth data.
 * @param authData The authentication data for the object.
 * @param userData Details about the user, such as ID and role.
 * @param validator Contains methods for permission checks like isAdmin, isDeleted, and isPublic.
 * @param inputsById Optional data for additional permission logic context.
 * @returns A promise that resolves to an object with permission keys and boolean values.
 */
export async function calculatePermissions<T extends AuthDataItem>(
    authData: T,
    userData: Pick<SessionUser, "id"> | null,
    validator: {
        owner: (data: Record<string, unknown>, userId: string | null) => RawOwnerData | null;
        isDeleted: (data: Record<string, unknown>) => boolean;
        isPublic: (data: Record<string, unknown>, getParentInfo?: ((id: string, typename: ModelType | `${ModelType}`) => Record<string, unknown> | undefined)) => boolean;
        permissionResolvers: (params: {
            isAdmin: boolean;
            isDeleted: boolean;
            isLoggedIn: boolean;
            isPublic: boolean;
            data: Record<string, unknown>;
            userId: string | null;
        }) => Record<string, () => any>;
    },
    inputsById?: InputsById,
): Promise<ResolvedPermissions> {
    // If the user is an admin in relation to the object (i.e. they can do anything to it). This DOES NOT mean they're a site admin!
    const ownerData = validator.owner(authData, userData?.id ?? null);
    // Convert bigint IDs to strings for OwnerData compatibility
    const convertedOwnerData: OwnerData = {
        Team: ownerData?.Team ? {
            id: ownerData.Team.id.toString(),
            members: ownerData.Team.members?.map(member => ({
                userId: member.userId.toString(),
                isAdmin: member.isAdmin,
            })),
        } : null,
        User: ownerData?.User ? { id: ownerData.User.id.toString() } : null,
    };
    const isAdmin = userData?.id ? isOwnerAdminCheck(convertedOwnerData, userData.id) : false;
    const isDeleted = validator.isDeleted(authData);
    const isLoggedIn = !!userData?.id;
    const isPublic = validator.isPublic(authData, inputsById ? (id: string, typename: `${ModelType}`) => getParentInfo(id, typename, inputsById) : undefined);

    const permissionResolvers = validator.permissionResolvers({ isAdmin, isDeleted, isLoggedIn, isPublic, data: authData, userId: userData?.id ?? null });

    // Execute resolvers and convert to object entries
    const permissions = await Promise.all(
        Object.entries(permissionResolvers).map(async ([key, resolver]) => [key, await resolver()]),
    ).then(entries => Object.fromEntries(entries));

    return permissions;
}

/**
 * Type for permissions result object
 */
type PermissionsById = Record<string, ResolvedPermissions>;

/**
 * Using queried permissions data, calculates permissions for multiple objects. These are used to let a user know ahead of time if they're allowed to 
 * perform an action, and indicate that in the UX
 * @param authDataById Map of all queried data required to validate permissions, keyed by ID.
 * @param inputsById Map of all input data, keyed by ID
 * @param userData Data about the user performing the action
 * @returns Map of permissions objects, keyed by ID
 */
export async function getMultiTypePermissions(
    authDataById: AuthDataById,
    inputsById: InputsById,
    userData: Pick<SessionUser, "id"> | null,
): Promise<PermissionsById> {
    // Initialize result
    const permissionsById: PermissionsById = {};
    // Loop through each ID and calculate permissions
    for (const [id, authData] of Object.entries(authDataById)) {
        // Get permissions object for this ID
        if (!hasTypename(authData)) {
            throw new CustomError("0033", "InternalError", { authData });
        }
        const modelLogic = ModelMap.get(authData.__typename);
        if (!modelLogic || !modelLogic.validate) {
            throw new CustomError("0034", "InternalError", { type: authData.__typename });
        }
        const validator = modelLogic.validate();
        permissionsById[id] = await calculatePermissions(authData, userData, validator, inputsById);
    }
    return permissionsById;
}

/**
 * Using object type and ids, calculate permissions for one object type
 * @param type Object type
 * @param ids IDs of objects to calculate permissions for
 * @param userData Data about the user performing the action
 * @returns Permissions object, where each field is a permission and each value is an array
 * of that permission's value for each object (in the same order as the IDs)
 */
export async function getSingleTypePermissions<Permissions extends Record<string, boolean>>(
    type: `${ModelType}`,
    ids: string[],
    userData: Pick<SessionUser, "id" | "languages"> | null,
): Promise<{ [K in keyof Permissions]: Permissions[K][] }> {
    // Initialize result
    const permissions: Partial<{ [K in keyof Permissions]: Permissions[K][] }> = {};
    // Get validator and prismaDelegate
    const { dbTable, validate } = ModelMap.getLogic(["dbTable", "validate"], type, true, "getSingleTypePermissions");
    const validator = validate();
    // Get auth data for all objects
    let select: Record<string, unknown> | undefined;
    let dataById: Record<string, object>;
    try {
        select = permissionsSelectHelper(validator.permissionsSelect, userData?.id ?? null);
        const authData = await (DbProvider.get()[dbTable] as PrismaDelegate).findMany({
            where: { id: { in: ids } },
            select,
        });
        dataById = Object.fromEntries(authData.map(item => [item.id, item as AuthDataItem]));
    } catch (error) {
        throw new CustomError("0388", "InternalError", { ids, select: select ?? "undefined", objectType: type });
    }
    // Loop through each id and calculate permissions
    for (const id of ids) {
        // Get permissions object for this ID
        const permissionsObject = await calculatePermissions(dataById[id] as AuthDataItem, userData, validator);
        // Add permissions object to result
        for (const key of Object.keys(permissionsObject)) {
            permissions[key as keyof Permissions] = [...(permissions[key as keyof Permissions] ?? []), permissionsObject[key]];
        }
    }
    return permissions as { [K in keyof Permissions]: Permissions[K][] };
}

/**
 * Type for auth data objects with typename
 */
type TypedAuthData = AuthDataItem & { __typename: `${ModelType}` };

/**
 * Validates that the user has permission to perform one or more actions on the given objects. Throws 
 * an error if the user does not have permission.
 * @param authDataById Map of all queried data required to validate permissions, keyed by ID.
 * @param idsByAction Map of object IDs to validate permissions for, keyed by action. We store actions this way (instead of keyed by ID) 
 * in case one ID is used for multiple actions.
 * @param inputsById Map of all input data, keyed by ID
 * @param userData Data about the user performing the action
 * @param throwsOnError Whether to throw an error if the user does not have permission, or return a boolean
 */
export async function permissionsCheck(
    authDataById: Record<string, TypedAuthData>,
    idsByAction: { [key in QueryAction]?: string[] },
    inputsById: InputsById,
    userData: Pick<SessionUser, "id"> | null,
    throwsOnError = true,
): Promise<boolean> {
    const adminId = await DbProvider.getAdminId();
    const isAdmin = userData?.id && userData.id === adminId;
    // Get permissions
    const permissionsById = await getMultiTypePermissions(authDataById, inputsById, userData);
    // If you're an admin, skip permissions checking
    if (isAdmin) {
        return true;
    }
    // Loop through each action and validate permissions
    for (const action of Object.keys(idsByAction)) {
        // Skip "Create" action
        if (action === "Create") continue;
        // Get IDs for this action
        const ids = idsByAction[action];
        // Loop through each ID and validate permissions
        for (const id of ids) {
            // Skip placeholder IDs
            if (id === DUMMY_ID) continue;
            // Skip if the ID appears in the "Create" action (e.g. connecting to a chat which is also being created in the same request). 
            // NOTE: This opens the possibility for an attack where a user tries to create an object using another object's ID, so that the connect
            // permissions check is sipped. However, as long as we use a transaction for the full request (i.e. if that object fails to create, 
            // the whole request fails), this should be fine.
            if (idsByAction.Create?.includes(id)) continue;
            // Get permissions for this ID
            const permissions = permissionsById[id];
            // If permissions doesn't exist, something went wrong.
            if (!permissions) {
                if (throwsOnError) {
                    throw new CustomError("0390", "CouldNotFindPermissions", { action, id, __typename: authDataById?.[id]?.__typename });
                } else {
                    return false;
                }
            }
            // Check if permissions contains the current action. If so, make sure it's not false.
            if (`can${action}` in permissions && !permissions[`can${action}`]) {
                if (throwsOnError) {
                    throw new CustomError("0297", "Unauthorized", { action, id, __typename: authDataById[id].__typename });
                } else {
                    return false;
                }
            }
        }
    }
    return true;
}
