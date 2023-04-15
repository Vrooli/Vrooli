export const Api_list = `fragment Api_list on Api {
versions {
    translations {
        id
        language
        details
        name
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