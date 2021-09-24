import { gql } from 'graphql-tag';

export const customerSessionFields = gql`
    fragment customerSessionFields on Customer {
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