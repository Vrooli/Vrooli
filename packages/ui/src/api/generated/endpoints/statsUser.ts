import gql from 'graphql-tag';

export const statsUserFindMany = gql`
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
            projectsCompletionTimeAverage
            quizzesPassed
            quizzesFailed
            routines
            routinesCompleted
            routinesCompletionTimeAverage
            runsStarted
            runsCompleted
            runsCompletionTimeAverage
            smartContractsCreated
            smartContractsCompleted
            smartContractsCompletionTimeAverage
            standardsCreated
            standardsCompleted
            standardsCompletionTimeAverage
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

