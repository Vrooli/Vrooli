import { gql } from 'graphql-tag';

export const sessionFields = gql`
    fragment sessionFields on User {
        id
        theme
        roles {
            title
            description
        }
    }
`