import gql from 'graphql-tag';

export const tagFindMany = gql`
query tags($input: TagSearchInput!) {
  tags(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            tag
            bookmarks
            translations {
                id
                language
                description
            }
            you {
                isOwn
                isBookmarked
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

