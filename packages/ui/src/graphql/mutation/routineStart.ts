import { gql } from 'graphql-tag';
import { logFields } from 'graphql/fragment';

export const routineStartMutation = gql`
    ${logFields}
    mutation routineStart($input: RoutineStartInput!) {
        routineStart(input: $input) {
            ...logFields
        }
    }
`