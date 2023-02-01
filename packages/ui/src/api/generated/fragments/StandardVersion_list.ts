export const StandardVersion_list = `fragment StandardVersion_list on StandardVersion {
translations {
    id
    language
    description
    jsonVariable
}
id
created_at
updated_at
isComplete
isFile
isLatest
isPrivate
default
standardType
props
yup
versionIndex
versionLabel
commentsCount
directoryListingsCount
forksCount
reportsCount
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