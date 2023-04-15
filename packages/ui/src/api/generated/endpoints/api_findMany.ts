import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

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
                canReact
                isBookmarked
                isViewed
                reaction
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

