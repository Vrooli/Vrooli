export const PullRequest_list = `fragment PullRequest_list on PullRequest {
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
    isBot
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
