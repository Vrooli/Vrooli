import gql from 'graphql-tag';

export const quizFindOne = gql`
query quiz($input: FindByIdInput!) {
  quiz(input: $input) {
    quizQuestions {
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
        translations {
            id
            language
            helpText
            questionText
        }
        id
        created_at
        updated_at
        order
        points
        responsesCount
        standardVersion {
            id
            isLatest
            isPrivate
            versionIndex
            versionLabel
            root {
                id
                isPrivate
            }
            translations {
                id
                language
                description
                jsonVariable
            }
        }
        you {
            canDelete
            canEdit
        }
    }
    stats {
        id
        created_at
        periodStart
        periodEnd
        periodType
        timesStarted
        timesPassed
        timesFailed
        scoreAverage
        completionTimeAverage
    }
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
  }
}`;

export const quizFindMany = gql`
query quizzes($input: QuizSearchInput!) {
  quizzes(input: $input) {
    edges {
        cursor
        node {
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
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const quizCreate = gql`
mutation quizCreate($input: QuizCreateInput!) {
  quizCreate(input: $input) {
    quizQuestions {
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
        translations {
            id
            language
            helpText
            questionText
        }
        id
        created_at
        updated_at
        order
        points
        responsesCount
        standardVersion {
            id
            isLatest
            isPrivate
            versionIndex
            versionLabel
            root {
                id
                isPrivate
            }
            translations {
                id
                language
                description
                jsonVariable
            }
        }
        you {
            canDelete
            canEdit
        }
    }
    stats {
        id
        created_at
        periodStart
        periodEnd
        periodType
        timesStarted
        timesPassed
        timesFailed
        scoreAverage
        completionTimeAverage
    }
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
  }
}`;

export const quizUpdate = gql`
mutation quizUpdate($input: QuizUpdateInput!) {
  quizUpdate(input: $input) {
    quizQuestions {
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
        translations {
            id
            language
            helpText
            questionText
        }
        id
        created_at
        updated_at
        order
        points
        responsesCount
        standardVersion {
            id
            isLatest
            isPrivate
            versionIndex
            versionLabel
            root {
                id
                isPrivate
            }
            translations {
                id
                language
                description
                jsonVariable
            }
        }
        you {
            canDelete
            canEdit
        }
    }
    stats {
        id
        created_at
        periodStart
        periodEnd
        periodType
        timesStarted
        timesPassed
        timesFailed
        scoreAverage
        completionTimeAverage
    }
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
  }
}`;

