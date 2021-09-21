import { gql } from 'graphql-tag';

export const profileQuery = gql`
    query {
        profile {
            id
            firstName
            lastName
            pronouns
            theme
            emails {
                id
                emailAddress
                receivesDeliveryUpdates
            }
        }
    }
`