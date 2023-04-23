export const RoutineVersionInput_full = `fragment RoutineVersionInput_full on RoutineVersionInput {
id
index
isRequired
name
standardVersion {
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
