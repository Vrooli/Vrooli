import gql from 'graphql-tag';

export const pullRequestCreate = gql`
mutation pullRequestCreate($input: PullRequestCreateInput!) {
  pullRequestCreate(input: $input) {
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

