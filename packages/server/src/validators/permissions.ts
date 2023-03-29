import { GqlModelType, SessionUser } from '@shared/consts';
import { permissionsSelectHelper } from "../builders";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { PrismaType } from "../types";
import { QueryAction } from "../utils/types";
import { isOwnerAdminCheck } from "./isOwnerAdminCheck";

/**
 * Handles setting and interpreting permission policies for organizations and their objects. Permissions are stored as stringified JSON 
 * in the database (in case we change the structure later), which means we must query for permissions ahead of a crud operation, and then validate the
 * permissions against the operation.
 * 
 * NOTE: When an object is owned by a user instead of an organization, there is no permissioning required. The user is the owner, and can do whatever they want.
 * 
 * Objects which have permissions are: 
 * - Organization
 * - Member (of an organization)
 * - Role (of an organization)
 * - Project
 * - Routine
 * - Smart Contract
 * - Standard
 * 
 * Multiple permissions must be checked, depending on the object type and its relationship to the user. Permissions can also override each other. 
 * Here is every object type and its permissions hierarchy (from most important to least important):
 * - Organization -> Organization
 * - Member -> Organization
 * - Role -> Organization
 * - Project -> Member, Role, Project, Organization
 * - Routine -> Member, Role, Routine, Organization
 * - Smart Contract -> Member, Role, Smart Contract, Organization
 * - Standard -> Member, Role, Standard, Organization
 * 
 * Example: You try to update a project which belongs to an organization. First we check if the organization permits this (is not deleted, not locked, etc). 
 * Then we check if you have a role in the organization, and if that role allows you to update projects. Then we check if you have a membership in the organization 
 * and if that membership explicitly excludes you from updating projects. Then we check if the project itself allows you to update it (is not deleted, not locked, etc). 
 * It's a lot of checks, but it's necessary to ensure that permissions are enforced correctly.
 */
export type PermissionType = 'Create' | 'Read' | 'Update' | 'Delete' | 'Fork' | 'Report' | 'Run';

/**
 * Permissions policy fields which are common to all policies in an organization. All related to how and how often the policy can be updated.
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
}

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
}

/**
 * Permissions policy fields which are common to all nested policies in an organization.
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
}

/**
 * Permissions policy for entire organization. Organization itself can have same fields in OrganizationPolicyPart, but can also 
 * specify more specific permissions that apply to parts of the organization.
 */
export type OrganizationPolicy = PolicyPart & {
    /**
     * Policy schema version. Used to help us read JSON properly and migrate policy shapes in the future, if necessary.
     */
    version: number;
    members?: NestedPolicyPart & {
        /**
         * One or more routines which a user must complete before they can be added to the organization.
         */
        onboardingRoutineVersionIds?: string[];
        /**
         * One or more projects which a user must complete before they can be added to the organization.
         */
        onboardingProjectIds?: string[];
        /**
         * One or more quizzes which a user must complete before they can be added to the organization.
         */
        onboardingQuizIds?: string[];
    };
    roles?: NestedPolicyPart;
    info?: NestedPolicyPart;
    issues?: NestedPolicyPart;
    notes?: NestedPolicyPart;
    projects?: NestedPolicyPart;
    questions?: NestedPolicyPart;
    quizzes?: NestedPolicyPart;
    routines?: NestedPolicyPart;
    smartContracts?: NestedPolicyPart;
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
}

/**
 * Permissions policy for a specific member of an organization. Useful to lock down a member's permissions, or to
 * give them special permissions.
 */
export type MemberPolicy = {
    version: number;
    /**
     * Gives admin privileges. Admins can do anything, including change the organization's policy.
     */
    isAdmin?: boolean;
    isLocked?: boolean;
    lockedUntil?: Date;
}

/**
 * Permissions policy for a specific role of an organization. Useful to lock down a role's permissions, or to
 * give them special permissions.
 */
export type RolePolicy = {
    version: number;
    /**
     * Gives admin privileges. Admins can do anything, including change the organization's policy.
     */
    isAdmin?: boolean;
    isLocked?: boolean;
    lockedUntil?: Date;
}

/**
 * Permissions policy for a specific project of an organization. Useful to lock down a project's permissions
 */
export type ProjectPolicy = {
    version: number;
    whoCanAdd?: MemberRolePolicy;
    whoCanDelete?: MemberRolePolicy;
    whoCanUpdate?: MemberRolePolicy;
}

/**
 * Permissions policy for a specific routine of an organization. Useful to lock down a routine's permissions
 */
export type RoutinePolicy = {
    version: number;
    whoCanAdd?: MemberRolePolicy;
    whoCanDelete?: MemberRolePolicy;
    whoCanUpdate?: MemberRolePolicy;
}

/**
 * Permissions policy for a specific smart contract of an organization. Useful to lock down a smart contract's permissions
 */
export type SmartContractPolicy = {
    version: number;
    whoCanAdd?: MemberRolePolicy;
    whoCanDelete?: MemberRolePolicy;
    whoCanUpdate?: MemberRolePolicy;
}

/**
 * Permissions policy for a specific standard of an organization. Useful to lock down a standard's permissions
 */
export type StandardPolicy = {
    version: number;
    whoCanAdd?: MemberRolePolicy;
    whoCanDelete?: MemberRolePolicy;
    whoCanUpdate?: MemberRolePolicy;
}

/**
 * Using queried permissions data, calculates permissions for multiple objects. These are used to let a user know ahead of time if they're allowed to 
 * perform an action, and indicate that in the UX
 * @param authDataById Map of all queried data required to validate permissions, keyed by ID.
 * @parma userData Data about the user performing the action
 * @returns Map of permissions objects, keyed by ID
 */
export async function getMultiTypePermissions(
    authDataById: { [id: string]: { __typename: `${GqlModelType}`, [x: string]: any } },
    userData: SessionUser | null,
): Promise<{ [id: string]: { [x: string]: any } }> {
    // Initialize result
    const permissionsById: { [id: string]: { [key in QueryAction]?: boolean } } = {};
    // Loop through each ID and calculate permissions
    for (const id of Object.keys(authDataById)) {
        // Get permissions object for this ID
        const { validate } = getLogic(['validate'], authDataById[id].__typename, userData?.languages ?? ['en'], 'getMultiplePermissions');
        const isAdmin = isOwnerAdminCheck(validate.owner(authDataById[id]), userData?.id);
        const isDeleted = validate.isDeleted(authDataById[id], userData?.languages ?? ['en']);
        const isPublic = validate.isPublic(authDataById[id], userData?.languages ?? ['en']);
        const permissionResolvers = validate.permissionResolvers({ isAdmin, isDeleted, isPublic, data: authDataById[id] });
        // permissionResolvers is an object of key/resolver pairs. We want to create a new object with 
        // the same keys, but with the values of the resolvers instead.
        const permissions = await Promise.all(
            Object.entries(permissionResolvers).map(async ([key, resolver]) => [key, await resolver()])
        ).then(entries => Object.fromEntries(entries));
        // Add permissions object to result
        permissionsById[id] = permissions;
    }
    return permissionsById;
}

/**
 * Using object type and ids, calculate permissions for one object type
 * @param type Object type
 * @param ids IDs of objects to calculate permissions for
 * @param prisma Prisma client
 * @param userData Data about the user performing the action
 * @returns Permissions object, where each field is a permission and each value is an array
 * of that permission's value for each object (in the same order as the IDs)
 */
export async function getSingleTypePermissions<Permissions extends { [x: string]: any }>(
    type: `${GqlModelType}`,
    ids: string[],
    prisma: PrismaType,
    userData: SessionUser | null,
): Promise<{ [K in keyof Permissions]: Permissions[K][] }> {
    // Initialize result
    const permissions: Partial<{ [K in keyof Permissions]: Permissions[K][] }> = {};
    // Get validator and prismaDelegate
    const { delegate, validate } = getLogic(['delegate', 'validate'], type, userData?.languages ?? ['en'], 'getSingleTypePermissions');
    // Get auth data for all objects
    let select: any;
    let authData: any = [];
    try {
        select = permissionsSelectHelper(validate.permissionsSelect, userData?.id ?? null, userData?.languages ?? ['en']);
        authData = await delegate(prisma).findMany({
            where: { id: { in: ids } },
            select,
        })
    } catch (error) {
        throw new CustomError('0388', 'InternalError', userData?.languages ?? ['en'], { ids, select, objectType: type });
    }
    // Loop through each object and calculate permissions
    for (const authDataItem of authData) {
        const isAdmin = isOwnerAdminCheck(validate.owner(authDataItem), userData?.id);
        const isDeleted = validate.isDeleted(authDataItem, userData?.languages ?? ['en']);
        const isPublic = validate.isPublic(authDataItem, userData?.languages ?? ['en']);
        const permissionResolvers = validate.permissionResolvers({ isAdmin, isDeleted, isPublic, data: authDataItem });
        // permissionResolvers is an array of key/resolver pairs. We can use this to create an object with the same keys
        // as the permissions object, but with the values being the result of the resolver.
        const permissionsObject = Object.fromEntries(Object.entries(permissionResolvers).map(([key, resolver]) => [key, resolver()]));
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
 * @parma userId ID of user requesting permissions
 */
export async function permissionsCheck(
    authDataById: { [id: string]: { __typename: `${GqlModelType}`, [x: string]: any } },
    idsByAction: { [key in QueryAction]?: string[] },
    userData: SessionUser | null,
) {
    // Get permissions
    const permissionsById = await getMultiTypePermissions(authDataById, userData);
    // Loop through each action and validate permissions
    for (const action of Object.keys(idsByAction)) {
        // Get IDs for this action
        const ids = idsByAction[action];
        // Loop through each ID and validate permissions
        for (const id of ids) {
            // Get permissions for this ID
            const permissions = permissionsById[id];
            // Make sure permissions exists. If not, and not create, something went wrong.
            if (!permissions) {
                if (action !== 'Create') {
                    throw new CustomError('0390', 'InternalError', userData?.languages ?? ['en'], { action, id, __typename: authDataById[id].__typename });
                }
                continue;
            }
            // Check if permissions contains the current action. If so, make sure it's not false.
            if (`can${action}` in permissions && !permissions[`can${action}`]) {
                throw new CustomError('0297', 'Unauthorized', userData?.languages ?? ['en'], { action, id, __typename: authDataById[id].__typename });
            }
        }
    }
}