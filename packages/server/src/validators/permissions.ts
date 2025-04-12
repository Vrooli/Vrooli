import { DUMMY_ID, ModelType, SEEDED_IDS, SessionUser } from "@local/shared";
import { permissionsSelectHelper } from "../builders/permissionsSelectHelper.js";
import { PrismaDelegate } from "../builders/types.js";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { ModelMap } from "../models/base/index.js";
import { Validator } from "../models/types.js";
import { AuthDataById, AuthDataItem } from "../utils/getAuthenticatedData.js";
import { InputsById, QueryAction } from "../utils/types.js";
import { isOwnerAdminCheck } from "./isOwnerAdminCheck.js";

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
        /**
         * One or more quizzes which a user must complete before they can be added to the team.
         */
        onboardingQuizIds?: string[];
    };
    roles?: NestedPolicyPart;
    info?: NestedPolicyPart;
    codes?: NestedPolicyPart;
    issues?: NestedPolicyPart;
    notes?: NestedPolicyPart;
    projects?: NestedPolicyPart;
    questions?: NestedPolicyPart;
    quizzes?: NestedPolicyPart;
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
export function getParentInfo(id: string, typename: `${ModelType}`, inputsById: InputsById): any | undefined {
    const node = inputsById[id]?.node;
    if (node?.__typename !== typename) return undefined;
    return node?.parent ? inputsById[node.parent.id]?.input : undefined;
}

/**
 * Calculates and returns permissions for a given object based on user and auth data.
 * @param authData The authentication data for the object.
 * @param userData Details about the user, such as ID and role.
 * @param validator Contains methods for permission checks like isAdmin, isDeleted, and isPublic.
 * @param inputsById Optional data for additional permission logic context.
 * @returns A promise that resolves to an object with permission keys and boolean values.
 */
export async function calculatePermissions(
    authData: AuthDataItem,
    userData: Pick<SessionUser, "id"> | null,
    validator: ReturnType<Validator<any>>,
    inputsById?: InputsById,
): Promise<ResolvedPermissions> {
    const isAdmin = userData?.id ? isOwnerAdminCheck(validator.owner(authData, userData.id), userData.id) : false;
    const isDeleted = validator.isDeleted(authData);
    const isLoggedIn = !!userData?.id;
    const isPublic = validator.isPublic(authData, (...rest) => inputsById ? getParentInfo(...rest, inputsById) : undefined);

    const permissionResolvers = validator.permissionResolvers({ isAdmin, isDeleted, isLoggedIn, isPublic, data: authData, userId: userData?.id });

    // Execute resolvers and convert to object entries
    const permissions = await Promise.all(
        Object.entries(permissionResolvers).map(async ([key, resolver]) => [key, await resolver()]),
    ).then(entries => Object.fromEntries(entries));

    return permissions;
}

/**
 * Using queried permissions data, calculates permissions for multiple objects. These are used to let a user know ahead of time if they're allowed to 
 * perform an action, and indicate that in the UX
 * @param authDataById Map of all queried data required to validate permissions, keyed by ID.
 * @param userData Data about the user performing the action
 * @returns Map of permissions objects, keyed by ID
 */
export async function getMultiTypePermissions(
    authDataById: AuthDataById,
    inputsById: InputsById,
    userData: Pick<SessionUser, "id"> | null,
): Promise<{ [id: string]: { [x: string]: any } }> {
    // Initialize result
    const permissionsById: { [id: string]: { [key in QueryAction as `can${key}`]?: boolean } } = {};
    // Loop through each ID and calculate permissions
    for (const [id, authData] of Object.entries(authDataById)) {
        // Get permissions object for this ID
        const validator = ModelMap.get(authData.__typename).validate();
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
export async function getSingleTypePermissions<Permissions extends { [x: string]: any }>(
    type: `${ModelType}`,
    ids: string[],
    userData: Pick<SessionUser, "id" | "languages"> | null,
): Promise<{ [K in keyof Permissions]: Permissions[K][] }> {
    // Initialize result
    const permissions: Partial<{ [K in keyof Permissions]: Permissions[K][] }> = {};
    // Get validator and prismaDelegate
    const { dbTable, validate } = ModelMap.getLogic(["dbTable", "validate"], type);
    const validator = validate();
    // Get auth data for all objects
    let select: any;
    let dataById: Record<string, object>;
    try {
        select = permissionsSelectHelper(validator.permissionsSelect, userData?.id ?? null);
        const authData = await (DbProvider.get()[dbTable] as PrismaDelegate).findMany({
            where: { id: { in: ids } },
            select,
        });
        dataById = Object.fromEntries(authData.map(item => [item.id, item]));
    } catch (error) {
        throw new CustomError("0388", "InternalError", { ids, select, objectType: type });
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
 * Validates that the user has permission to perform one or more actions on the given objects. Throws 
 * an error if the user does not have permission.
 * @param authDataById Map of all queried data required to validate permissions, keyed by ID.
 * @param idsByAction Map of object IDs to validate permissions for, keyed by action. We store actions this way (instead of keyed by ID) 
 * in case one ID is used for multiple actions.
 * @param userId ID of user requesting permissions
 * @param throwsOnError Whether to throw an error if the user does not have permission, or return a boolean
 */
export async function permissionsCheck(
    authDataById: { [id: string]: { __typename: `${ModelType}`, [x: string]: any } },
    idsByAction: { [key in QueryAction]?: string[] },
    inputsById: InputsById,
    userData: Pick<SessionUser, "id"> | null,
    throwsOnError = true,
): Promise<boolean> {
    // Get permissions
    const permissionsById = await getMultiTypePermissions(authDataById, inputsById, userData);
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
                    // If you're ad admin, do it anyway
                    const isAdmin = userData?.id && userData.id === SEEDED_IDS.User.Admin;
                    if (isAdmin) {
                        logger.warning("Using admin privileges to perform action", { action, id, __typename: authDataById[id].__typename });
                    } else {
                        throw new CustomError("0297", "Unauthorized", { action, id, __typename: authDataById[id].__typename });
                    }
                } else {
                    return false;
                }
            }
        }
    }
    return true;
}
