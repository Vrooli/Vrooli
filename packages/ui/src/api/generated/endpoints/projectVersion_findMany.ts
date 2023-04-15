import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const projectVersionFindMany = gql`${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

query projectVersions($input: ProjectVersionSearchInput!) {
  projectVersions(input: $input) {
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
            runProjectsCount
            simplicity
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

