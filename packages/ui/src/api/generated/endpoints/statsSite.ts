import gql from 'graphql-tag';

export const statsSiteFindMany = gql`
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
            projectsCompletionTimeAverage
            quizzes
            quizzesCompleted
            quizScoreAverage
            routines
            routinesCompleted
            routinesCompletionTimeAverage
            routinesSimplicityAverage
            routinesComplexityAverage
            runsStarted
            runsCompleted
            runsCompletionTimeAverageInPerid
            smartContractsCreated
            smartContractsCompleted
            smartContractsCompletionTimeAverage
            smartContractCalls
            standardsCreated
            standardsCompleted
            standardsCompletionTimeAverage
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

