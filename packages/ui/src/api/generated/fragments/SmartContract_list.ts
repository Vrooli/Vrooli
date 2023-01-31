export const SmartContract_list = `fragment SmartContract_list on SmartContract {
versions {
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
        canEdit
        canReport
        canUse
        canView
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
    canEdit
    canStar
    canTransfer
    canView
    canVote
    isStarred
    isUpvoted
    isViewed
}
}`;