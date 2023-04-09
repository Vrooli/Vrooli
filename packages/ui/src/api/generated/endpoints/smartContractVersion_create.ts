import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const smartContractVersionCreate = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation smartContractVersionCreate($input: SmartContractVersionCreateInput!) {
  smartContractVersionCreate(input: $input) {
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
  }
}`;

