import { gql } from 'graphql-tag';

export const routineProgressUpdateMutation = gql`
    mutation routineProgressUpdate($input: RoutineProgressUpdateInput!) {
        routineProgressUpdate(input: $input) {
            success
        }
    }
`