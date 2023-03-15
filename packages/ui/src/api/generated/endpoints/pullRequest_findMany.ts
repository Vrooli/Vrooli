import gql from 'graphql-tag';

export const pullRequestFindMany = gql`
query pullRequests($input: PullRequestSearchInput!) {
  pullRequests(input: $input) {
    edges {
        cursor
        node {
            translations {
                id
                language
                text
            }
            id
            created_at
            updated_at
            mergedOrRejectedAt
            commentsCount
            status
            createdBy {
                id
                name
                handle
            }
            you {
                canComment
                canDelete
                canReport
                canUpdate
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

