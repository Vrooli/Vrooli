import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const smartContractFindOne = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

query smartContract($input: FindByIdInput!) {
  smartContract(input: $input) {
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
    versions {
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
}`;

export const smartContractFindMany = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

query smartContracts($input: SmartContractSearchInput!) {
  smartContracts(input: $input) {
    edges {
        cursor
        node {
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
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const smartContractCreate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation smartContractCreate($input: SmartContractCreateInput!) {
  smartContractCreate(input: $input) {
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
    versions {
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
}`;

export const smartContractUpdate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation smartContractUpdate($input: SmartContractUpdateInput!) {
  smartContractUpdate(input: $input) {
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
    versions {
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
}`;

