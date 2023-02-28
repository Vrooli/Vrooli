import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

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
