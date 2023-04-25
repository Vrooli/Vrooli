export const QuestionAnswer_common = `fragment QuestionAnswer_common on QuestionAnswer {
id
created_at
updated_at
createdBy {
    id
    name
    handle
}
score
bookmarks
isAccepted
commentsCount
}`;
