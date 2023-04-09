import gql from 'graphql-tag';

export const userFindMany = gql`
query users($input: UserSearchInput!) {
  users(input: $input) {
    edges {
        cursor
        node {
            translations {
                id
                language
                bio
            }
            id
            created_at
            handle
            name
            bookmarks
            reportsReceivedCount
            you {
                canDelete
                canReport
                canUpdate
                isBookmarked
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

