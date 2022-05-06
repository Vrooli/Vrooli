import { gql } from 'graphql-tag';
import { listRunFields, listStarFields, listViewFields } from 'graphql/fragment';

export const forYouPageQuery = gql`
    ${listRunFields}
    ${listStarFields}
    ${listViewFields}
    query forYouPage($input: ForYouPageInput!) {
        forYouPage(input: $input) {
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