import { gql } from 'graphql-tag';
import { listProjectFields, listRoutineFields } from 'graphql/fragment';

export const learnQuery = gql`
    ${listProjectFields}
    ${listRoutineFields}
    query learn {
        learn {
            courses {
                ...listProjectFields
            }
            tutorials {
                ...listRoutineFields
            }
        }
    }
`