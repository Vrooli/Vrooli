import { gql } from 'graphql-tag';
import { runRoutineFields } from 'graphql/fragment';

export const runRoutineCompleteMutation = gql`
    ${runRoutineFields}
    mutation runRoutineComplete($input: RunRoutineCompleteInput!) {
        runRoutineComplete(input: $input) {
            ...runRoutineFields
        }
    }
`