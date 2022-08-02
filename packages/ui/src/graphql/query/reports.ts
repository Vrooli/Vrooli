import { gql } from 'graphql-tag';
import { reportFields } from 'graphql/fragment';

export const reportsQuery = gql`
    ${reportFields}
    query reports($input: ReportSearchInput!) {
        reports(input: $input) {
            pageInfo {
                hasNextPage
                endCursor
            }
            edges {
                cursor
                node {
                    ...reportFields
                }
            }
        }
    }
`