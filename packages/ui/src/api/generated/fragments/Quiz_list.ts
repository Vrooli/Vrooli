export const Quiz_list = `fragment Quiz_list on Quiz {
translations {
    id
    language
    description
    name
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
attemptsCount
quizQuestionsCount
project {
    id
    isPrivate
}
routine {
    id
    isInternal
    isPrivate
}
you {
    canDelete
    canBookmark
    canUpdate
    canRead
    canReact
    hasCompleted
    isBookmarked
    reaction
}
}`;
