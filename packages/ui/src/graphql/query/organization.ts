import { gql } from 'graphql-tag';
import { organizationFields } from 'graphql/fragment';

export const organizationQuery = gql`
    ${organizationFields}
    query organization($input: FindByIdOrHandleInput!) {
        organization(input: $input) {
            ...organizationFields
        }
    }
`