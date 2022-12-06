import { gql } from 'graphql-tag';
import { runRoutineFields } from 'graphql/fragment';

export const runRoutineCreateMutation = gql`
    ${runRoutineFields}
    mutation runRoutineCreate($input: RunRoutineCreateInput!) {
        runRoutineCreate(input: $input) {
            ...runRoutineFields
        }
    }
`