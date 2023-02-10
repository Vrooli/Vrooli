import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const smartContractVersionFindOne = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

query smartContractVersion($input: FindByIdInput!) {
  smartContractVersion(input: $input) {
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
            created_at
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
  }
}`;

export const smartContractVersionFindMany = gql`
query smartContractVersions($input: SmartContractVersionSearchInput!) {
  smartContractVersions(input: $input) {
    edges {
        cursor
        node {
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
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const smartContractVersionCreate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation smartContractVersionCreate($input: SmartContractVersionCreateInput!) {
  smartContractVersionCreate(input: $input) {
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
            created_at
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
  }
}`;

export const smartContractVersionUpdate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation smartContractVersionUpdate($input: SmartContractVersionUpdateInput!) {
  smartContractVersionUpdate(input: $input) {
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
            created_at
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
  }
}`;

