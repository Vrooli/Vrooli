import { gql } from 'graphql-tag';
import { runFields } from 'graphql/fragment';

export const runCompleteMutation = gql`
    ${runFields}
    mutation runComplete($input: RunCompleteInput!) {
        runComplete(input: $input) {
            ...runFields
        }
    }
`