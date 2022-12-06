import { gql } from 'graphql-tag';
import { listRunRoutineFields, listStarFields, listViewFields } from 'graphql/fragment';

export const historyQuery = gql`
    ${listRunRoutineFields}
    ${listStarFields}
    ${listViewFields}
    query history($input: HistoryInput!) {
        history(input: $input) {
            activeRuns {
                ...listRunRoutineFields
            }
            completedRuns {
                ...listRunRoutineFields
            }
            recentlyViewed {
                ...listViewFields
            }
            recentlyStarred {
                ...listStarFields
            }
        }
    }
`