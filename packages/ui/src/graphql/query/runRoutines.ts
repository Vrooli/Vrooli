import { gql } from 'graphql-tag';
import { listRunRoutineFields } from 'graphql/fragment';

export const runRoutinesQuery = gql`
    ${listRunRoutineFields}
    query runRoutines($input: RunRoutineSearchInput!) {
        runRoutines(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listRunRoutineFields
                }
            }
        }
    }
`