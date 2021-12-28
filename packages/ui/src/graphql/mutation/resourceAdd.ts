import { gql } from 'graphql-tag';
import { resourceFields } from 'graphql/fragment';

export const resourceAddMutation = gql`
    ${resourceFields}
    mutation resourceAdd($input: ResourceInput!) {
        resourceAdd(input: $input) {
            ...resourceFields
        }
    }
`