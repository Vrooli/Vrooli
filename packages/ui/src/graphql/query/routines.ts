import { gql } from 'graphql-tag';
import { routineFields } from 'graphql/fragment';

export const routinesQuery = gql`
    ${routineFields}
    query routines($input: RoutinesQueryInput!) {
        routines(input: $input) {
            ...routineFields
        }
    }
`