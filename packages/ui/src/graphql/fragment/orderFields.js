import { gql } from 'graphql-tag';

export const orderFields = gql`
    fragment orderFields on Order {
        id
        status
        specialInstructions
        desiredDeliveryDate
        expectedDeliveryDate
        isDelivery
        address {
            tag
            name
            country
            administrativeArea
            subAdministrativeArea
            locality
            postalCode
            throughfare
            premise
        }
        customer {
            id
        }
    }
`