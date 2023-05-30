export const Routine_list = `fragment Routine_list on Routine {
versions {
    translations {
        id
        language
        description
        instructions
        name
    }
    id
    created_at
    updated_at
    completedAt
    isAutomatable
    isComplete
    isDeleted
    isLatest
    isPrivate
    simplicity
    timesStarted
    timesCompleted
    smartContractCallData
    apiCallData
    versionIndex
    versionLabel
    commentsCount
    directoryListingsCount
    forksCount
    inputsCount
    nodesCount
    nodeLinksCount
    outputsCount
    reportsCount
}
id
created_at
updated_at
isInternal
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
    canComment
    canDelete
    canBookmark
    canUpdate
    canRead
    canReact
    isBookmarked
    isViewed
    reaction
}
}`;
