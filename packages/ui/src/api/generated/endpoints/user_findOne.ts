import gql from "graphql-tag";

export const userFindOne = gql`
query user($input: FindByIdOrHandleInput!) {
  user(input: $input) {
    botSettings
    translations {
        id
        language
        bio
    }
    id
    created_at
    handle
    isBot
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

