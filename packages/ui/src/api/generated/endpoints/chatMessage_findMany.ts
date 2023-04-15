import gql from 'graphql-tag';

export const chatMessageFindMany = gql`
query chatMessages($input: ChatMessageSearchInput!) {
  chatMessages(input: $input) {
    edges {
        cursor
        node {
            translations {
                id
                language
                text
            }
            id
            created_at
            updated_at
            user {
                id
                name
                handle
            }
            score
            reportsCount
            you {
                canDelete
                canReply
                canReport
                canUpdate
                canReact
                reaction
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

