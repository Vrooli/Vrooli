import gql from "graphql-tag";

export const quizQuestionFindMany = gql`
query quizQuestions($input: QuizQuestionSearchInput!) {
  quizQuestions(input: $input) {
    edges {
        cursor
        node {
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

