import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const standardVersionFindOne = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

query standardVersion($input: FindByIdInput!) {
  standardVersion(input: $input) {
    versionNotes
    pullRequest {
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
            canEdit
            canReport
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
            }
        }
        stats {
            id
            created_at
            periodStart
            periodEnd
            periodType
            linksToInputs
            linksToOutputs
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
    }
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
    isFile
    isLatest
    isPrivate
    default
    standardType
    props
    yup
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
}`;

export const standardVersionFindMany = gql`
query standardVersions($input: StandardVersionSearchInput!) {
  standardVersions(input: $input) {
    edges {
        cursor
        node {
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
            isFile
            isLatest
            isPrivate
            default
            standardType
            props
            yup
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const standardVersionCreate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation standardVersionCreate($input: StandardVersionCreateInput!) {
  standardVersionCreate(input: $input) {
    versionNotes
    pullRequest {
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
            canEdit
            canReport
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
            }
        }
        stats {
            id
            created_at
            periodStart
            periodEnd
            periodType
            linksToInputs
            linksToOutputs
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
    }
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
    isFile
    isLatest
    isPrivate
    default
    standardType
    props
    yup
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
}`;

export const standardVersionUpdate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation standardVersionUpdate($input: StandardVersionUpdateInput!) {
  standardVersionUpdate(input: $input) {
    versionNotes
    pullRequest {
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
            canEdit
            canReport
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
            }
        }
        stats {
            id
            created_at
            periodStart
            periodEnd
            periodType
            linksToInputs
            linksToOutputs
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
    }
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
    isFile
    isLatest
    isPrivate
    default
    standardType
    props
    yup
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
}`;

