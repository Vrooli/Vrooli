import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const apiFindOne = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

query api($input: FindByIdInput!) {
  api(input: $input) {
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
    versions {
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
}`;

export const apiFindMany = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

query apis($input: ApiSearchInput!) {
  apis(input: $input) {
    edges {
        cursor
        node {
            versions {
                translations {
                    id
                    language
                    details
                    name
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
                    canReport
                    canUpdate
                    canUse
                    canRead
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const apiCreate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation apiCreate($input: ApiCreateInput!) {
  apiCreate(input: $input) {
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
    versions {
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
}`;

export const apiUpdate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation apiUpdate($input: ApiUpdateInput!) {
  apiUpdate(input: $input) {
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
    versions {
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
}`;

