import { gql } from 'graphql-tag';
import { logFields } from 'graphql/fragment';

export const routineCancelMutation = gql`
    ${logFields}
    mutation routineCancel($input: RoutineCancelInput!) {
        routineCancel(input: $input) {
            ...logFields
        }
    }
`