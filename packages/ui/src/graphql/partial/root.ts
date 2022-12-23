export const rootPermissionFields = ['RootPermission', `{
    canDelete
    canEdit
    canStar
    canTranswer
    canView
    canVote
}`] as const;
export const versionPermissionFields = ['VersionPermission', `{
    canComment
    canCopy
    canDelete
    canEdit
    canReport
    canUse
    canView
}`] as const;