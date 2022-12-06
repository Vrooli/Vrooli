import { gql } from 'graphql-tag';
import { runRoutineFields } from 'graphql/fragment';

export const runRoutineUpdateMutation = gql`
    ${runRoutineFields}
    mutation runRoutineUpdate($input: RunRoutineUpdateInput!) {
        runRoutineUpdate(input: $input) {
            ...runRoutineFields
        }
    }
`