import gql from 'graphql-tag';

export const quizAttemptFindOne = gql`
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
}`;

