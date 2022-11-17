import { organizationValidator } from "../organization"
import { userValidator } from "../user"

/**
 * Checks if the user has admin privileges on an object's creator/owner
 */
export const isOwnerAdminCheck = <PrismaData extends { [x: string]: any}>(
    data: PrismaData,
    organization: (data: PrismaData) => any,
    user: (data: PrismaData) => any,
    userId: string | null,
): boolean => {
    if (userId === null) {
        return false
    }
    const organizationData = organization(data)
    const ownerData = user(data)
    return userValidator().isAdmin(ownerData, userId) || organizationValidator().isAdmin(organizationData, userId);
}