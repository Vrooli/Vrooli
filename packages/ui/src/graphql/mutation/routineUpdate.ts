import { gql } from 'graphql-tag';
import { routineFields } from 'graphql/fragment';

export const routineUpdateMutation = gql`
    ${routineFields}
    mutation routineUpdate($input: RoutineInput!) {
        routineUpdate(input: $input) {
            ...routineFields
        }
    }
`