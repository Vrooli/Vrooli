import { gql } from 'graphql-tag';
import { resourceFields } from 'graphql/fragment';

export const resourceAddMutation = gql`
    ${resourceFields}
    mutation resourceAdd($input: ResourceAddInput!) {
        resourceAdd(input: $input) {
            ...resourceFields
        }
    }
`