import gql from 'graphql-tag';

export const pullRequestUpdate = gql`
mutation pullRequestUpdate($input: PullRequestUpdateInput!) {
  pullRequestUpdate(input: $input) {
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
}`;

