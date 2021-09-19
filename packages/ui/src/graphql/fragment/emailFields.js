import { gql } from 'graphql-tag';

export const emailFields = gql`
    fragment emailFields on Email {
        id
        emailAddress
        receivesDeliveryUpdates
    }
`