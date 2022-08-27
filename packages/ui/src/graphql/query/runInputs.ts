import { gql } from 'graphql-tag';
import { runInputFields } from 'graphql/fragment';

export const runInputsQuery = gql`
    ${runInputFields}
    query runInputs($input: RunInputSearchInput!) {
        runInputs(input: $input) {
            ...runInputFields
        }
    }
`