import gql from 'graphql-tag';

export const userFindOne = gql`
query user($input: FindByIdOrHandleInput!) {
  user(input: $input) {
    stats {
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
    translations {
        id
        language
        bio
    }
    id
    created_at
    handle
    name
    bookmarks
    reportsReceivedCount
    you {
        canDelete
        canReport
        canUpdate
        isBookmarked
        isViewed
    }
  }
}`;

