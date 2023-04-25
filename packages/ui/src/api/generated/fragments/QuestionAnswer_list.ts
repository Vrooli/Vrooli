export const QuestionAnswer_list = `fragment QuestionAnswer_list on QuestionAnswer {
translations {
    id
    language
    text
}
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
