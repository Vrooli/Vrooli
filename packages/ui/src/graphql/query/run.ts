import { gql } from 'graphql-tag';
import { runFields } from 'graphql/fragment';

export const runQuery = gql`
    ${runFields}
    query run($input: FindByIdInput!) {
        run(input: $input) {
            ...runFields
        }
    }
`