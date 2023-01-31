import gql from 'graphql-tag';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Tag_list } from '../fragments/Tag_list';
import { Label_list } from '../fragments/Label_list';

export const smartContractFindOne = gql`...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}

query smartContract($input: FindByIdInput!) {
  smartContract(input: $input) {
    stats {
        id
        created_at
        periodStart
        periodEnd
        periodType
        calls
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

export const smartContractFindMany = gql`...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}

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

export const smartContractCreate = gql`...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}

mutation smartContractCreate($input: SmartContractCreateInput!) {
  smartContractCreate(input: $input) {
    stats {
        id
        created_at
        periodStart
        periodEnd
        periodType
        calls
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

export const smartContractUpdate = gql`...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}

mutation smartContractUpdate($input: SmartContractUpdateInput!) {
  smartContractUpdate(input: $input) {
    stats {
        id
        created_at
        periodStart
        periodEnd
        periodType
        calls
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

