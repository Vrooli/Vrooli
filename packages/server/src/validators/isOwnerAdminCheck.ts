/**
 * Checks if the user has admin privileges on an object's creator/owner
 */
export const isOwnerAdminCheck = (
    owner: {
        Organization?: { [x: string]: any } | null,
        User?: { [x: string]: any } | null,
    },
    userId: string | null | undefined,
): boolean => {
    console.log('isOwnerAdminCheck', JSON.stringify(owner), userId);
    // Can't be an admin if not logged in
    if (userId === null) return false;
    // If the owner is a user, check id
    if (owner.User) return owner.User.id === userId;
    // If the owner is an organization, check if you're a member with "isAdmin" set to true
    if (owner.Organization) {
        return owner.Organization.members ? 
            owner.Organization.members.some((member: { [x: string]: any }) => member.userId === userId && member.isAdmin) :
            false;
    }
    // If the owner is neither a user nor an organization, return false
    return false;
}