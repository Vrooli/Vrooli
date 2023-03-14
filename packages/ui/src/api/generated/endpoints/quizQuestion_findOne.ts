import gql from 'graphql-tag';

export const quizQuestionFindOne = gql`
query quizQuestion($input: FindByIdInput!) {
  quizQuestion(input: $input) {
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
        }
    }
    you {
        canDelete
        canUpdate
    }
  }
}`;

