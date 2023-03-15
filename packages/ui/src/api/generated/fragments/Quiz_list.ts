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
views
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
    canVote
    hasCompleted
    isBookmarked
    isUpvoted
}
}`;