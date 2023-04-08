import gql from 'graphql-tag';

export const userFindOne = gql`
query user($input: FindByIdOrHandleInput!) {
  user(input: $input) {
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
}`;

