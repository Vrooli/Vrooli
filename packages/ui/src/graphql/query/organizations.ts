import { gql } from 'graphql-tag';
import { organizationFields } from 'graphql/fragment';

export const organizationsQuery = gql`
    ${organizationFields}
    query organizations($input: OrganizationsQueryInput!) {
        organizations(input: $input) {
            ...organizationFields
        }
    }
`