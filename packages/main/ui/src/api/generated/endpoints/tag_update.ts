import gql from "graphql-tag";

export const tagUpdate = gql`
mutation tagUpdate($input: TagUpdateInput!) {
  tagUpdate(input: $input) {
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
}`;

