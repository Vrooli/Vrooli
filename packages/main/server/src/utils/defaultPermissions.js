export const defaultPermissions = ({ isAdmin, isDeleted, isLoggedIn, isPublic }) => ({
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
});
//# sourceMappingURL=defaultPermissions.js.map