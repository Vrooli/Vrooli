import gql from 'graphql-tag';

export const statsRoutineFindMany = gql`
query statsRoutine($input: StatsRoutineSearchInput!) {
  statsRoutine(input: $input) {
    edges {
        cursor
        node {
            id
            periodStart
            periodEnd
            periodType
            runsStarted
            runsCompleted
            runCompletionTimeAverage
            runContextSwitchesAverage
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

