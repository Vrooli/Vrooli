import { gql } from 'graphql-tag';
import { runFields } from 'graphql/fragment';

export const runCancelMutation = gql`
    ${runFields}
    mutation runCancel($input: RunCancelInput!) {
        runCancel(input: $input) {
            ...runFields
        }
    }
`