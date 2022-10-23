import { gql } from 'graphql-tag';

export const sessionFields = gql`
    fragment sessionFields on Session {
        isLoggedIn
        users {
            handle
            id
            languages
            name
            theme
        }
    }
`