import { gql } from 'graphql-tag';
import { listPullRequestFields } from 'graphql/fragment';

export const pullRequestsQuery = gql`
    ${listPullRequestFields}
    query pullRequests($input: PullRequestSearchInput!) {
        pullRequests(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listPullRequestFields
                }
            }
        }
    }
`