export const RoutineVersionOutput_full = `fragment RoutineVersionOutput_full on RoutineVersionOutput {
id
index
name
standardVersion {
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
        canReport
        canUpdate
        canUse
        canRead
    }
}
translations {
    id
    language
    description
    helpText
}
}`;