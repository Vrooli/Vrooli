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
isCompleted
score
stars
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
    canEdit
    canStar
    canView
    canVote
    isStarred
    isUpvoted
}
}`;