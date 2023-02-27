import gql from 'graphql-tag';

export const tagFindOne = gql`
query tag($input: FindByIdInput!) {
  tag(input: $input) {
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

