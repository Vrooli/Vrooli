import { gql } from 'graphql-tag';
import { routineFields } from 'graphql/fragment';

export const routineCreateMutation = gql`
    ${routineFields}
    mutation routineCreate($input: RoutineCreateInput!) {
        routineCreate(input: $input) {
            ...routineFields
        }
    }
`