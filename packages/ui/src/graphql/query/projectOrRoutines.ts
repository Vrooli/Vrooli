import { gql } from 'graphql-tag';
import { listProjectFields, listRoutineFields } from 'graphql/fragment';

export const projectOrRoutinesQuery = gql`
    ${listProjectFields}
    ${listRoutineFields}
    query projectOrRoutines($input: ProjectOrRoutineSearchInput!) {
        projectOrRoutines(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ... on Project {
                        ...listProjectFields
                    }
                    ... on Routine {
                        ...listRoutineFields
                    }
                }
            }
        }
    }
`