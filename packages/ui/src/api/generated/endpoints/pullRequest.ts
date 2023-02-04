import gql from 'graphql-tag';

export const pullRequestFindOne = gql`
query pullRequest($input: FindByIdInput!) {
  pullRequest(input: $input) {
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
        canEdit
        canReport
    }
  }
}`;

export const pullRequestFindMany = gql`
query pullRequests($input: PullRequestSearchInput!) {
  pullRequests(input: $input) {
    edges {
        cursor
        node {
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
                canEdit
                canReport
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const pullRequestCreate = gql`
mutation pullRequestCreate($input: PullRequestCreateInput!) {
  pullRequestCreate(input: $input) {
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
        canEdit
        canReport
    }
  }
}`;

export const pullRequestUpdate = gql`
mutation pullRequestUpdate($input: PullRequestUpdateInput!) {
  pullRequestUpdate(input: $input) {
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
        canEdit
        canReport
    }
  }
}`;

export const pullRequestAccept = gql`
mutation pullRequestAcdept($input: FindByIdInput!) {
  pullRequestAcdept(input: $input) {
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
        canEdit
        canReport
    }
  }
}`;

export const pullRequestReject = gql`
mutation pullRequestReject($input: FindByIdInput!) {
  pullRequestReject(input: $input) {
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
        canEdit
        canReport
    }
  }
}`;

