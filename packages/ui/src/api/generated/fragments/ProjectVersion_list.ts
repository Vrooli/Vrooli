export const ProjectVersion_list = `fragment ProjectVersion_list on ProjectVersion {
directories {
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    childOrder
    isRoot
    projectVersion {
        id
        isLatest
        isPrivate
        versionIndex
        versionLabel
        root {
            id
            isPrivate
        }
        translations {
            id
            language
            description
            name
        }
    }
}
translations {
    id
    language
    description
    name
}
id
created_at
updated_at
directoriesCount
isLatest
isPrivate
reportsCount
runsCount
simplicity
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