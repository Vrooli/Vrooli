import gql from "graphql-tag";

export const bookmarkListFindMany = gql`
query bookmarkLists($input: BookmarkListSearchInput!) {
  bookmarkLists(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            updated_at
            label
            bookmarksCount
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

