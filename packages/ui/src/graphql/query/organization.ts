import { gql } from 'graphql-tag';
import { organizationFields } from 'graphql/fragment';

export const organizationQuery = gql`
    ${organizationFields}
    query organization($input: FindByIdInput!) {
        organization(input: $input) {
            ...organizationFields
        }
    }
`