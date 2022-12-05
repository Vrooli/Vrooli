import { gql } from 'graphql-tag';
import { listProjectFields, listRoutineFields } from 'graphql/fragment';

export const developQuery = gql`
    ${listProjectFields}
    ${listRoutineFields}
    query develop {
        develop {
            completed {
                ... on Project {
                    ...listProjectFields
                }
                ... on Routine {
                    ...listRoutineFields
                }
            }
            inProgress {
                ... on Project {
                    ...listProjectFields
                }
                ... on Routine {
                    ...listRoutineFields
                }
            }
            recent {
                ... on Project {
                    ...listProjectFields
                }
                ... on Routine {
                    ...listRoutineFields
                }
            }
        }
    }
`