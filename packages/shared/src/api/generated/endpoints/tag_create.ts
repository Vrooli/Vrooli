import gql from "graphql-tag";

export const tagCreate = gql`
mutation tagCreate($input: TagCreateInput!) {
  tagCreate(input: $input) {
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

