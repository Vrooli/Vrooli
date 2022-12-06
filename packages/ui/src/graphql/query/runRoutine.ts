import { gql } from 'graphql-tag';
import { runRoutineFields } from 'graphql/fragment';

export const runRoutineQuery = gql`
    ${runRoutineFields}
    query runRoutine($input: FindByIdInput!) {
        runRoutine(input: $input) {
            ...runRoutineFields
        }
    }
`