import gql from "graphql-tag";
export const statsQuizFindMany = gql `
query statsQuiz($input: StatsQuizSearchInput!) {
  statsQuiz(input: $input) {
    edges {
        cursor
        node {
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;
//# sourceMappingURL=statsQuiz_findMany.js.map