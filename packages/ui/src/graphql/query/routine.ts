import { gql } from 'graphql-tag';
import { routineFields } from 'graphql/fragment';

export const routineQuery = gql`
    ${routineFields}
    query routine($input: FindByIdInput!) {
        routine(input: $input) {
            ...routineFields
        }
    }
`