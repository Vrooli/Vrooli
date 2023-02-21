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

export const pullRequestAccept = gql`
mutation pullRequestAcdept($input: FindByIdInput!) {
  pullRequestAcdept(input: $input) {
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

export const pullRequestReject = gql`
mutation pullRequestReject($input: FindByIdInput!) {
  pullRequestReject(input: $input) {
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

