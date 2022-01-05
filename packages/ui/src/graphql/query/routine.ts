import { gql } from 'graphql-tag';
import { deepRoutineFields } from 'graphql/fragment';

export const routineQuery = gql`
    ${deepRoutineFields}
    query routine($input: FindByIdInput!) {
        routine(input: $input) {
            ...deepRoutineFields
        }
    }
`