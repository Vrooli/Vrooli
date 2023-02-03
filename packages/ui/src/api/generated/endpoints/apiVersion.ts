import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const apiVersionFindOne = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

query apiVersion($input: FindByIdInput!) {
  apiVersion(input: $input) {
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
                summary
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
    translations {
        id
        language
        details
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
        canEdit
        canReport
        canUse
        canView
    }
  }
}`;

export const apiVersionFindMany = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

query apiVersions($input: ApiVersionSearchInput!) {
  apiVersions(input: $input) {
    edges {
        cursor
        node {
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
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const apiVersionCreate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation apiVersionCreate($input: ApiVersionCreateInput!) {
  apiVersionCreate(input: $input) {
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
                summary
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
    translations {
        id
        language
        details
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
        canEdit
        canReport
        canUse
        canView
    }
  }
}`;

export const apiVersionUpdate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation apiVersionUpdate($input: ApiVersionUpdateInput!) {
  apiVersionUpdate(input: $input) {
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
                summary
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
    translations {
        id
        language
        details
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
        canEdit
        canReport
        canUse
        canView
    }
  }
}`;

