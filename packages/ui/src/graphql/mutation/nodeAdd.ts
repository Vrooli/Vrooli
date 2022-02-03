import { gql } from 'graphql-tag';
import { nodeFields } from 'graphql/fragment';

export const nodeAddMutation = gql`
    ${nodeFields}
    mutation nodeAdd($input: NodeAddInput!) {
        nodeAdd(input: $input) {
            ...nodeFields
        }
    }
`