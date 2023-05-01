export const Routine_full = `fragment Routine_full on Routine {
parent {
    id
    isAutomatable
    isComplete
    isDeleted
    isLatest
    isPrivate
    root {
        id
        isInternal
        isPrivate
    }
    translations {
        id
        language
        description
        instructions
        name
    }
    versionIndex
    versionLabel
}
stats {
    id
    periodStart
    periodEnd
    periodType
    runsStarted
    runsCompleted
    runCompletionTimeAverage
    runContextSwitchesAverage
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
