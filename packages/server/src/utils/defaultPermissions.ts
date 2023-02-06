/**
 * Holds common permission resolvers, which can apply to most models. 
 * Some models may have custom resolvers, which can easily be set in 
 * their ModelLogic object.
 */
export const defaultPermissions = ({ isAdmin, isDeleted, isPublic }: { isAdmin: boolean, isDeleted: boolean, isPublic: boolean }) => ({
    canComment: () => !isDeleted && (isAdmin || isPublic),
    canConnect: () => !isDeleted && (isAdmin || isPublic),
    canCopy: () => !isDeleted && (isAdmin || isPublic),
    canDelete: () => !isDeleted && isAdmin,
    canDisconnect: () => true,
    canRead: () => !isDeleted && (isPublic || isAdmin),
    canReport: () => !isAdmin && !isDeleted && isPublic,
    canRun: () => !isDeleted && (isAdmin || isPublic),
    canStar: () => !isDeleted && (isAdmin || isPublic),
    canTransfer: () => isAdmin && !isDeleted,
    canUpdate: () => !isDeleted && isAdmin,
    canUse: () => !isDeleted && (isAdmin || isPublic),
    canVote: () => !isDeleted && (isAdmin || isPublic),
})