export const SmartContractVersion_full = `fragment SmartContractVersion_full on SmartContractVersion {
versionNotes
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
resourceList {
    id
    created_at
    translations {
        id
        language
        description
        name
    }
    resources {
        id
        index
        link
        usedFor
        translations {
            id
            language
            description
            name
        }
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
            description
            jsonVariable
            name
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
    description
    jsonVariable
    name
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
    canReport
    canUpdate
    canUse
    canRead
}
}`;