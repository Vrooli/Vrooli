import { gql } from 'graphql-tag';
import { listProjectFields, listRoutineFields } from 'graphql/fragment';

export const developPageQuery = gql`
    ${listProjectFields}
    ${listRoutineFields}
    query developPage {
        developPage {
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