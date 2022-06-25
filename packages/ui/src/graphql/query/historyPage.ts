import { gql } from 'graphql-tag';
import { listRunFields, listStarFields, listViewFields } from 'graphql/fragment';

export const historyPageQuery = gql`
    ${listRunFields}
    ${listStarFields}
    ${listViewFields}
    query historyPage($input: HistoryPageInput!) {
        historyPage(input: $input) {
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