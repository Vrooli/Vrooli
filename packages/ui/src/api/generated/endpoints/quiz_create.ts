import gql from "graphql-tag";

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
                }
                user {
                    id
                    name
                    handle
                }
                you {
                    canDelete
                    canUpdate
                }
            }
            you {
                canDelete
                canUpdate
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
                name
            }
        }
        you {
            canDelete
            canUpdate
        }
    }
    stats {
        id
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
  }
}`;

