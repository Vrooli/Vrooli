import gql from 'graphql-tag';

export const quizQuestionResponseCreate = gql`
mutation quizQuestionResponseCreate($input: QuizQuestionResponseCreateInput!) {
  quizQuestionResponseCreate(input: $input) {
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
    translations {
        id
        language
        response
    }
    you {
        canDelete
        canUpdate
    }
  }
}`;
