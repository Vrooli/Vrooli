import gql from "graphql-tag";
export const pullRequestReject = gql `
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
//# sourceMappingURL=pullRequest_reject.js.map