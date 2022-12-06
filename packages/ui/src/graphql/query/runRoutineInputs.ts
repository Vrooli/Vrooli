import { gql } from 'graphql-tag';
import { runRoutineInputFields } from 'graphql/fragment';

export const runRoutineInputsQuery = gql`
    ${runRoutineInputFields}
    query runRoutineInputs($input: RunRoutineInputSearchInput!) {
        runRoutineInputs(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...runRoutineInputFields
                }
            }
        }
    }
`