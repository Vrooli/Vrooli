import { gql } from 'graphql-tag';
import { routineFields } from 'graphql/fragment';

export const routinesQuery = gql`
    ${routineFields}
    query routines($input: RoutineSearchInput!) {
        routines(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...routineFields
                }
            }
        }
    }
`