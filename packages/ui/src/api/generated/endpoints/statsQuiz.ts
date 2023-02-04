import gql from 'graphql-tag';

export const statsQuizFindMany = gql`
query statsQuiz($input: StatsQuizSearchInput!) {
  statsQuiz(input: $input) {
    edges {
        cursor
        node {
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

