import { gql } from 'graphql-tag';
import { runFields } from 'graphql/fragment';

export const runCreateMutation = gql`
    ${runFields}
    mutation runCreate($input: RunCreateInput!) {
        runCreate(input: $input) {
            ...runFields
        }
    }
`