import { gql } from 'graphql-tag';
import { customerContactFields } from 'graphql/fragment';

export const customersQuery = gql`
    ${customerContactFields}
    query {
        customers {
            ...customerContactFields
            status
            roles {
                role {
                    title
                }
            }
        }
    }
`