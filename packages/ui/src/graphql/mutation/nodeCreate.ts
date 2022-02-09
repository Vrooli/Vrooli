import { gql } from 'graphql-tag';
import { nodeFields } from 'graphql/fragment';

export const nodeCreateMutation = gql`
    ${nodeFields}
    mutation nodeCreate($input: NodeCreateInput!) {
        nodeCreate(input: $input) {
            ...nodeFields
        }
    }
`