import { gql } from 'graphql-tag';
import { deepRoutineFields, projectFields } from 'graphql/fragment';

export const learnPageQuery = gql`
    ${deepRoutineFields}
    ${projectFields}
    query learnPage {
        learnPage {
            courses {
                ...projectFields
            }
            tutorials {
                ...deepRoutineFields
            }
        }
    }
`