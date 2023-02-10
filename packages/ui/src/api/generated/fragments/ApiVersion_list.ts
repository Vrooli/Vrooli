export const ApiVersion_list = `fragment ApiVersion_list on ApiVersion {
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
}
translations {
    id
    language
    details
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
    canEdit
    canReport
    canUse
    canView
}
}`;