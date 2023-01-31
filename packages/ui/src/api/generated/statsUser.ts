import gql from 'graphql-tag';

export const findMany = gql`
query statsUser($input: StatsUserSearchInput!) {
  statsUser(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            periodStart
            periodEnd
            periodType
            apis
            organizations
            projects
            projectsCompleted
            projectsCompletionTimeAverageInPeriod
            quizzesPassed
            quizzesFailed
            routines
            routinesCompleted
            routinesCompletionTimeAverageInPeriod
            runsStarted
            runsCompleted
            runsCompletionTimeAverageInPeriod
            smartContractsCreated
            smartContractsCompleted
            smartContractsCompletionTimeAverageInPeriod
            standardsCreated
            standardsCompleted
            standardsCompletionTimeAverageInPeriod
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

