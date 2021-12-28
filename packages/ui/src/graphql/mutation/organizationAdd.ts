import { gql } from 'graphql-tag';
import { organizationFields } from 'graphql/fragment';

export const organizationAddMutation = gql`
    ${organizationFields}
    mutation organizationAdd($input: OrganizationInput!) {
        organizationAdd(input: $input) {
            ...organizationFields
        }
    }
`