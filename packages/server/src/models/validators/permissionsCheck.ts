import { BasePermissions } from "../types";
import { PermissionsCheckProps } from "./types";

/**
 * Validates that the user has permission to perform one or more actions on an object. 
 * @returns True if the user has correct permisssions
 */
export async function permissionsCheck<PermissionObject extends BasePermissions>({
    actions,
    permissions,
}: PermissionsCheckProps<PermissionObject>): Promise<boolean> {
    for (const action of actions) {
        for (const permission of permissions) {
            if (!permission[action]) return false;
        }
    }
    return true;
}