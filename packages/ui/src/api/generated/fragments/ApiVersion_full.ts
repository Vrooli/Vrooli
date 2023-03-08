export const ApiVersion_full = `fragment ApiVersion_full on ApiVersion {
pullRequest {
    translations {
        id
        language
        text
    }
    id
    created_at
    updated_at
    mergedOrRejectedAt
    commentsCount
    status
    createdBy {
        id
        name
        handle
    }
    you {
        canComment
        canDelete
        canReport
        canUpdate
    }
}
root {
    parent {
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
            details
            name
            summary
        }
    }
    stats {
        id
        periodStart
        periodEnd
        periodType
        calls
        routineVersions
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
        canVote
        isBookmarked
        isUpvoted
        isViewed
    }
}
translations {
    id
    language
    details
    name
    summary
}
versionNotes
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
}`;