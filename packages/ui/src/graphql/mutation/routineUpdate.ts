import { gql } from 'graphql-tag';
import { deepRoutineFields } from 'graphql/fragment';

export const routineUpdateMutation = gql`
    ${deepRoutineFields}
    mutation routineUpdate($input: RoutineUpdateInput!) {
        routineUpdate(input: $input) {
            ...deepRoutineFields
        }
    }
`