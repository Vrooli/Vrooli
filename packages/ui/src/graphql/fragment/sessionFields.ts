import { gql } from 'graphql-tag';

export const sessionFields = gql`
    fragment sessionFields on Session {
        id
        theme
        roles {
            title
            description
        }
    }
`