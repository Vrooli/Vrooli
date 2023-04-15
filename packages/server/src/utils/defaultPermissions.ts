/**
 * Holds common permission resolvers, which can apply to most models. 
 * Some models may have custom resolvers, which can easily be set in 
 * their ModelLogic object.
 */
export const defaultPermissions = ({ isAdmin, isDeleted, isLoggedIn, isPublic }: { isAdmin: boolean, isDeleted: boolean, isLoggedIn: boolean, isPublic: boolean }) => ({
    canComment: () => isLoggedIn && !isDeleted && (isAdmin || isPublic),
    canConnect: () => isLoggedIn && !isDeleted && (isAdmin || isPublic),
    canCopy: () => isLoggedIn && !isDeleted && (isAdmin || isPublic),
    canDelete: () => isLoggedIn && !isDeleted && isAdmin,
    canDisconnect: () => isLoggedIn,
    canRead: () => !isDeleted && (isPublic || isAdmin),
    canReport: () => isLoggedIn && !isAdmin && !isDeleted && isPublic,
    canRun: () => !isDeleted && (isAdmin || isPublic),
    canBookmark: () => isLoggedIn && !isDeleted && (isAdmin || isPublic),
    canTransfer: () => isLoggedIn && isAdmin && !isDeleted,
    canUpdate: () => isLoggedIn && !isDeleted && isAdmin,
    canUse: () => isLoggedIn && !isDeleted && (isAdmin || isPublic),
    canReact: () => isLoggedIn && !isDeleted && (isAdmin || isPublic),
})