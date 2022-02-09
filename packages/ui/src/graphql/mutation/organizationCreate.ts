import { gql } from 'graphql-tag';
import { organizationFields } from 'graphql/fragment';

export const organizationCreateMutation = gql`
    ${organizationFields}
    mutation organizationCreate($input: OrganizationCreateInput!) {
        organizationCreate(input: $input) {
            ...organizationFields
        }
    }
`