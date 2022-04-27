import { gql } from 'graphql-tag';
import { listRoutineFields } from 'graphql/fragment';

export const routinesQuery = gql`
    ${listRoutineFields}
    query routines($input: RoutineSearchInput!) {
        routines(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listRoutineFields
                }
            }
        }
    }
`