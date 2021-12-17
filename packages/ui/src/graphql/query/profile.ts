import { gql } from 'graphql-tag';

export const profileQuery = gql`
    query profile {
        profile {
            id
            username
            theme
            emails {
                id
                emailAddress
                receivesAccountUpdates
                receivesBusinessUpdates
            }
        }
    }
`