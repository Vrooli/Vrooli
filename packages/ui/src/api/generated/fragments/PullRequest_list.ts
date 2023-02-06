export const PullRequest_list = `fragment PullRequest_list on PullRequest {
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
}`;