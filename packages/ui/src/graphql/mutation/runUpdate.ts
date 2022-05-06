import { gql } from 'graphql-tag';
import { runFields } from 'graphql/fragment';

export const runUpdateMutation = gql`
    ${runFields}
    mutation runUpdate($input: RunUpdateInput!) {
        runUpdate(input: $input) {
            ...runFields
        }
    }
`