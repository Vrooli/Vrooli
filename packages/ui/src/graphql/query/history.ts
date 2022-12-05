import { gql } from 'graphql-tag';
import { listRunFields, listStarFields, listViewFields } from 'graphql/fragment';

export const historyQuery = gql`
    ${listRunFields}
    ${listStarFields}
    ${listViewFields}
    query history($input: HistoryInput!) {
        history(input: $input) {
            activeRuns {
                ...listRunFields
            }
            completedRuns {
                ...listRunFields
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