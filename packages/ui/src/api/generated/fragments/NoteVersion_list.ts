export const NoteVersion_list = `fragment NoteVersion_list on NoteVersion {
translations {
    id
    language
    description
    text
}
id
created_at
updated_at
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