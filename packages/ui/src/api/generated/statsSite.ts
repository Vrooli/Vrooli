import gql from 'graphql-tag';

export const findMany = gql`
query statsSite($input: StatsSiteSearchInput!) {
  statsSite(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            periodStart
            periodEnd
            periodType
            activeUsers
            apiCallsPeriod
            apis
            organizations
            projects
            projectsCompleted
            projectsCompletionTimeAverageInPeriod
            quizzes
            quizzesCompleted
            quizScoreAverageInPeriod
            routines
            routinesCompleted
            routinesCompletionTimeAverageInPeriod
            routinesSimplicityAverage
            routinesComplexityAverage
            runsStarted
            runsCompleted
            runsCompletionTimeAverageInPerid
            smartContractsCreated
            smartContractsCompleted
            smartContractsCompletionTimeAverageInPeriod
            smartContractCalls
            standardsCreated
            standardsCompleted
            standardsCompletionTimeAverageInPeriod
            verifiedEmails
            verifiedWallets
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

