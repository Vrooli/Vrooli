export const Project_list = `fragment Project_list on Project {
versions {
    directories {
        translations {
            id
            language
            description
            name
        }
        id
        created_at
        updated_at
        childOrder
        isRoot
        projectVersion {
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
                name
            }
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