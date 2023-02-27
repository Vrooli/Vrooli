import gql from 'graphql-tag';

export const pullRequestFindOne = gql`
query pullRequest($input: FindByIdInput!) {
  pullRequest(input: $input) {
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

