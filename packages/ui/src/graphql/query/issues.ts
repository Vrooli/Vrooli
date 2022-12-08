import { gql } from 'graphql-tag';
import { listIssueFields } from 'graphql/fragment';

export const issuesQuery = gql`
    ${listIssueFields}
    query issues($input: IssueSearchInput!) {
        issues(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listIssueFields
                }
            }
        }
    }
`