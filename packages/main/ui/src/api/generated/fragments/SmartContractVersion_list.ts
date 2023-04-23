export const SmartContractVersion_list = `fragment SmartContractVersion_list on SmartContractVersion {
translations {
    id
    language
    description
    jsonVariable
    name
}
id
created_at
updated_at
isComplete
isDeleted
isLatest
isPrivate
default
contractType
content
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
    canReport
    canUpdate
    canUse
    canRead
}
}`;
