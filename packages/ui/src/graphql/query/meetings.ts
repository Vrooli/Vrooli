import { gql } from 'graphql-tag';
import { listMeetingFields } from 'graphql/fragment';

export const meetingsQuery = gql`
    ${listMeetingFields}
    query meetings($input: MeetingSearchInput!) {
        meetings(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listMeetingFields
                }
            }
        }
    }
`