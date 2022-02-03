import { gql } from 'graphql-tag';
import { nodeFields } from 'graphql/fragment';

export const nodeUpdateMutation = gql`
    ${nodeFields}
    mutation nodeUpdate($input: NodeUpdateInput!) {
        nodeUpdate(input: $input) {
            ...nodeFields
        }
    }
`