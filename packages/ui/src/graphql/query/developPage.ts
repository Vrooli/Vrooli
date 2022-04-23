import { gql } from 'graphql-tag';
import { deepRoutineFields, projectFields } from 'graphql/fragment';

export const developPageQuery = gql`
    ${deepRoutineFields}
    ${projectFields}
    query developPage {
        developPage {
            completed {
                ... on Project {
                    ...projectFields
                }
                ... on Routine {
                    ...deepRoutineFields
                }
            }
            inProgress {
                ... on Project {
                    ...projectFields
                }
                ... on Routine {
                    ...deepRoutineFields
                }
            }
            recent {
                ... on Project {
                    ...projectFields
                }
                ... on Routine {
                    ...deepRoutineFields
                }
            }
        }
    }
`