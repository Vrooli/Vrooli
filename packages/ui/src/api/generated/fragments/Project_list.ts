export const Project_list = `fragment Project_list on Project {
versions {
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
}
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
}`;