import gql from 'graphql-tag';

export const statsUserFindMany = gql`
query statsUser($input: StatsUserSearchInput!) {
  statsUser(input: $input) {
    edges {
        cursor
        node {
            id
            periodStart
            periodEnd
            periodType
            apisCreated
            organizationsCreated
            projectsCreated
            projectsCompleted
            projectCompletionTimeAverage
            quizzesPassed
            quizzesFailed
            routinesCreated
            routinesCompleted
            routineCompletionTimeAverage
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
            standardsCreated
            standardsCompleted
            standardCompletionTimeAverage
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

