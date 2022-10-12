import { gql } from 'graphql-tag';
import { routineFields } from 'graphql/fragment';

export const routineQuery = gql`
    ${routineFields}
    query routine($input: FindByVersionInput!) {
        routine(input: $input) {
            ...routineFields
        }
    }
`