import { gql } from 'graphql-tag';
import { organizationFields } from 'graphql/fragment';

export const organizationAddMutation = gql`
    ${organizationFields}
    mutation organizationAdd($input: OrganizationAddInput!) {
        organizationAdd(input: $input) {
            ...organizationFields
        }
    }
`