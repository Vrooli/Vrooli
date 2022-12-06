import { gql } from 'graphql-tag';
import { runRoutineFields } from 'graphql/fragment';

export const runRoutineCancelMutation = gql`
    ${runRoutineFields}
    mutation runRoutineCancel($input: RunRoutineCancelInput!) {
        runRoutineCancel(input: $input) {
            ...runRoutineFields
        }
    }
`