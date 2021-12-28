import { gql } from 'graphql-tag';
import { routineFields } from 'graphql/fragment';

export const routineAddMutation = gql`
    ${routineFields}
    mutation routineAdd($input: RoutineInput!) {
        routineAdd(input: $input) {
            ...routineFields
        }
    }
`