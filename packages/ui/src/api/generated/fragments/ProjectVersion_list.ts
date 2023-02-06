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
runsCount
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