import { gql } from 'graphql-tag';

export const userSessionFields = gql`
    fragment userSessionFields on User {
        id
        status
        theme
        roles {
            role {
                title
                description
            }
        }
    }
`