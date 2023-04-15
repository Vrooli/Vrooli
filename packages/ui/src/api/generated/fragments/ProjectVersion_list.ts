export const ProjectVersion_list = `fragment ProjectVersion_list on ProjectVersion {
root {
    id
    created_at
    updated_at
    isPrivate
    issuesCount
    labels {
        ...Label_list
    }
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    permissions
    questionsCount
    score
    bookmarks
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canDelete
        canBookmark
        canTransfer
        canUpdate
        canRead
        canReact
        isBookmarked
        isViewed
        reaction
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
runProjectsCount
simplicity
versionIndex
versionLabel
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