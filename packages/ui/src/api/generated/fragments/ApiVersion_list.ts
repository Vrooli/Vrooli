export const ApiVersion_list = `fragment ApiVersion_list on ApiVersion {
translations {
    id
    language
    details
    summary
}
id
created_at
updated_at
callLink
commentsCount
documentationLink
forksCount
isLatest
isPrivate
reportsCount
versionIndex
versionLabel
you {
    canComment
    canCopy
    canDelete
    canEdit
    canReport
    canUse
    canView
}
}`;