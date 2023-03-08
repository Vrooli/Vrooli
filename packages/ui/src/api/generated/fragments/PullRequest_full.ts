export const PullRequest_full = `fragment PullRequest_full on PullRequest {
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
}`;