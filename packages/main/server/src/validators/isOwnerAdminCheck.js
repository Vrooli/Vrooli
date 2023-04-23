export const isOwnerAdminCheck = (owner, userId) => {
    if (userId === null)
        return false;
    if (owner.User)
        return owner.User.id === userId;
    if (owner.Organization) {
        return owner.Organization.members ?
            owner.Organization.members.some((member) => member.userId === userId && member.isAdmin) :
            false;
    }
    return false;
};
//# sourceMappingURL=isOwnerAdminCheck.js.map