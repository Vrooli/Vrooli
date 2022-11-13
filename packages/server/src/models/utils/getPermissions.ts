import { BasePermissions } from "../types";
import { getValidatorAndDelegate } from "./getValidator";
import { GetPermissionsProps } from "./types";

/**
 * Finds all permissions for the given object ids
 */
export async function getPermissions<PermissionObject extends BasePermissions>({
    objectType,
    ids,
    prisma,
    userId,
}: GetPermissionsProps): Promise<PermissionObject[]> {
    // Find validator and prisma delegate for this object type
    const { validator, prismaDelegate } = getValidatorAndDelegate(objectType, prisma, 'getPermissions');
    // Get data required to calculate permissions
    const permissionsData = await prismaDelegate.findMany({
        where: { id: { in: ids } },
        select: validator.permissionsSelect,
    })
    // Calculate permissions for each object
    const permissions = permissionsData.map(x => validator.permissionsFromSelect(x, userId));
    return permissions;
}