import gql from 'graphql-tag';

export const memberFindMany = gql`
query members($input: MemberSearchInput!) {
  members(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            updated_at
            isAdmin
            permissions
            organization {
                id
                handle
                you {
                    canAddMembers
                    canDelete
                    canBookmark
                    canReport
                    canUpdate
                    canRead
                    isBookmarked
                    isViewed
                    yourMembership {
                        id
                        created_at
                        updated_at
                        isAdmin
                        permissions
                    }
                }
            }
            user {
                id
                name
                handle
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

