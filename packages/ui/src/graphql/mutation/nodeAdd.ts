import { gql } from 'graphql-tag';
import { nodeFields } from 'graphql/fragment';

export const nodeAddMutation = gql`
    ${nodeFields}
    mutation nodeAdd($input: NodeInput!) {
        nodeAdd(input: $input) {
            ...nodeFields
        }
    }
`