/**
 * Checks if the user has admin privileges on an object's creator/owner
 */
export function isOwnerAdminCheck(
    owner: {
        Team?: { [x: string]: any } | null,
        User?: { [x: string]: any } | null,
    },
    userId: string | null | undefined,
): boolean {
    // Can't be an admin if not logged in
    if (userId === null || userId === undefined) return false;
    // If the owner is a user, check id
    if (owner.User) return owner.User.id === userId;
    // If the owner is a team, check if you're a member with "isAdmin" set to true
    if (owner.Team) {
        return owner.Team.members ?
            owner.Team.members.some((member: { [x: string]: any }) => member.userId === userId && member.isAdmin) :
            false;
    }
    // If the owner is neither a user nor a team, return false
    return false;
}
