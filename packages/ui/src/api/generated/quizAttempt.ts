import gql from 'graphql-tag';

export const findOne = gql`
query quizAttempt($input: FindByIdInput!) {
  quizAttempt(input: $input) {
    responses {
        id
        created_at
        updated_at
        response
        quizAttempt {
            id
            created_at
            updated_at
            pointsEarned
            status
            contextSwitches
            timeTaken
            responsesCount
            quiz {
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
            }
            user {
                id
                name
                handle
            }
            you {
                canDelete
                canEdit
            }
        }
        translations {
            id
            language
            response
        }
        you {
            canDelete
            canEdit
        }
    }
    id
    created_at
    updated_at
    pointsEarned
    status
    contextSwitches
    timeTaken
    responsesCount
    quiz {
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
    }
    user {
        id
        name
        handle
    }
    you {
        canDelete
        canEdit
    }
  }
}`;

export const findMany = gql`
query quizAttempts($input: QuizAttemptSearchInput!) {
  quizAttempts(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            updated_at
            pointsEarned
            status
            contextSwitches
            timeTaken
            responsesCount
            quiz {
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
            }
            user {
                id
                name
                handle
            }
            you {
                canDelete
                canEdit
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const create = gql`
mutation quizAttemptCreate($input: QuizAttemptCreateInput!) {
  quizAttemptCreate(input: $input) {
    responses {
        id
        created_at
        updated_at
        response
        quizAttempt {
            id
            created_at
            updated_at
            pointsEarned
            status
            contextSwitches
            timeTaken
            responsesCount
            quiz {
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
            }
            user {
                id
                name
                handle
            }
            you {
                canDelete
                canEdit
            }
        }
        translations {
            id
            language
            response
        }
        you {
            canDelete
            canEdit
        }
    }
    id
    created_at
    updated_at
    pointsEarned
    status
    contextSwitches
    timeTaken
    responsesCount
    quiz {
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
    }
    user {
        id
        name
        handle
    }
    you {
        canDelete
        canEdit
    }
  }
}`;

export const update = gql`
mutation quizAttemptUpdate($input: QuizAttemptUpdateInput!) {
  quizAttemptUpdate(input: $input) {
    responses {
        id
        created_at
        updated_at
        response
        quizAttempt {
            id
            created_at
            updated_at
            pointsEarned
            status
            contextSwitches
            timeTaken
            responsesCount
            quiz {
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
            }
            user {
                id
                name
                handle
            }
            you {
                canDelete
                canEdit
            }
        }
        translations {
            id
            language
            response
        }
        you {
            canDelete
            canEdit
        }
    }
    id
    created_at
    updated_at
    pointsEarned
    status
    contextSwitches
    timeTaken
    responsesCount
    quiz {
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
    }
    user {
        id
        name
        handle
    }
    you {
        canDelete
        canEdit
    }
  }
}`;

