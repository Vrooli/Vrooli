import gql from 'graphql-tag';

export const quizQuestionResponseFindOne = gql`
query quizQuestionResponse($input: FindByIdInput!) {
  quizQuestionResponse(input: $input) {
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
}`;

