import { gql } from 'graphql-tag';
import { logFields } from 'graphql/fragment';

export const routineCompleteMutation = gql`
    ${logFields}
    mutation routineComplete($input: RoutineCompleteInput!) {
        routineComplete(input: $input) {
            ...logFields
        }
    }
`