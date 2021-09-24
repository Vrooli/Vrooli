import { gql } from 'graphql-tag';

export const profileQuery = gql`
    query {
        profile {
            id
            username
            theme
            emails {
                id
                emailAddress
                receivesDeliveryUpdates
            }
        }
    }
`