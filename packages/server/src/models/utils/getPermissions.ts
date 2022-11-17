import { getDelegate } from "./getDelegate";
import { getValidator } from "./getValidator";
import { GetPermissionsProps } from "./types";

/**
 * Finds all permissions for the given object ids
 */
export async function getPermissions<PermissionObject extends { [x: string]: any }>({
    objectType,
    ids,
    prisma,
    userId,
}: GetPermissionsProps): Promise<PermissionObject[]> {
    // Find validator and prisma delegate for this object type
    const validator = getValidator(objectType, 'getPermissions');
    const prismaDelegate = getDelegate(objectType, prisma, 'getPermissions');
    // Get data required to calculate permissions
    const permissionsData = await prismaDelegate.findMany({
        where: { id: { in: ids } },
        select: validator.permissionsSelect,
    })
    // Calculate permissions for each object
    const result = permissionsData.map((d: any) => {
        // Each validator has a list of resolvers to calculate each field of permissions
        const resolvers = validator.permissionResolvers(d, userId);
        // Initialize permissions object
        const permissions: { [x: string]: any } = {};
        for (let i = 0; i < resolvers.length; i++) {
            const resolver = resolvers[i];
            // Resolve each permission field
            permissions[resolver[0]] = resolver[1]();
        }
        return permissions;
    });
    return result;
}