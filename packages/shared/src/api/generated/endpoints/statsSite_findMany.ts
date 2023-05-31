import gql from "graphql-tag";

export const statsSiteFindMany = gql`
query statsSite($input: StatsSiteSearchInput!) {
  statsSite(input: $input) {
    edges {
        cursor
        node {
            id
            periodStart
            periodEnd
            periodType
            activeUsers
            apiCalls
            apisCreated
            organizationsCreated
            projectsCreated
            projectsCompleted
            projectCompletionTimeAverage
            quizzesCreated
            quizzesCompleted
            routinesCreated
            routinesCompleted
            routineCompletionTimeAverage
            routineSimplicityAverage
            routineComplexityAverage
            runProjectsStarted
            runProjectsCompleted
            runProjectCompletionTimeAverage
            runProjectContextSwitchesAverage
            runRoutinesStarted
            runRoutinesCompleted
            runRoutineCompletionTimeAverage
            runRoutineContextSwitchesAverage
            smartContractsCreated
            smartContractsCompleted
            smartContractCompletionTimeAverage
            smartContractCalls
            standardsCreated
            standardsCompleted
            standardCompletionTimeAverage
            verifiedEmailsCreated
            verifiedWalletsCreated
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

