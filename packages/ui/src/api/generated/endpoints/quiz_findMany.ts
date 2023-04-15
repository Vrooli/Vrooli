import gql from 'graphql-tag';

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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

