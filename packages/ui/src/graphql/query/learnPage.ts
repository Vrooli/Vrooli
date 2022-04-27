import { gql } from 'graphql-tag';
import { listProjectFields, listRoutineFields } from 'graphql/fragment';

export const learnPageQuery = gql`
    ${listProjectFields}
    ${listRoutineFields}
    query learnPage {
        learnPage {
            courses {
                ...listProjectFields
            }
            tutorials {
                ...listRoutineFields
            }
        }
    }
`