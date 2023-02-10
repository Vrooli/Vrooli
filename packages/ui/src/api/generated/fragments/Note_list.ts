export const Note_list = `fragment Note_list on Note {
versions {
    translations {
        id
        language
        description
        name
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
stars
tags {
    ...Tag_list
}
transfersCount
views
you {
    canDelete
    canStar
    canTransfer
    canUpdate
    canRead
    canVote
    isStarred
    isUpvoted
    isViewed
}
}`;