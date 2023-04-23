import gql from "graphql-tag";
export const pullRequestAccept = gql `
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
//# sourceMappingURL=pullRequest_accept.js.map