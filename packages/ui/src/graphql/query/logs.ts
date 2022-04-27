import { gql } from 'graphql-tag';
import { logFields } from 'graphql/fragment';

export const logsQuery = gql`
    ${logFields}
    query logs($input: LogSearchInput!) {
        logs(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...logFields
                }
            }
        }
    }
`