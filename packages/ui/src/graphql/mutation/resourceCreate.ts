import { gql } from 'graphql-tag';
import { resourceFields } from 'graphql/fragment';

export const resourceCreateMutation = gql`
    ${resourceFields}
    mutation resourceCreate($input: ResourceCreateInput!) {
        resourceCreate(input: $input) {
            ...resourceFields
        }
    }
`