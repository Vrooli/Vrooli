import gql from 'graphql-tag';

export const findMany = gql`
query statsRoutine($input: StatsRoutineSearchInput!) {
  statsRoutine(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            periodStart
            periodEnd
            periodType
            runsStarted
            runsCompleted
            runCompletionTimeAverageInPeriod
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

