import { gql } from 'graphql-tag';
import { organizationFields, standardFields, userFields } from 'graphql/fragment';

export const standardQuery = gql`
    ${organizationFields}
    ${standardFields}
    ${userFields}
    query standard($input: FindByIdInput!) {
        standard(input: $input) {
            ...standardFields
            creator {
                ... on User {
                    ...userFields
                }
                ... on Organization {
                    ...organizationFields
                }
            }
        }
    }
`