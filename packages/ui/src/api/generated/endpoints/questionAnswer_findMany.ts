import gql from 'graphql-tag';

export const questionAnswerFindMany = gql`
query questionAnswers($input: QuestionAnswerSearchInput!) {
  questionAnswers(input: $input) {
    edges {
        cursor
        node {
            translations {
                id
                language
                text
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
            isAccepted
            commentsCount
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

