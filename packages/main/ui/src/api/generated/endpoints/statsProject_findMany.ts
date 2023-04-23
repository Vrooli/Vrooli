import gql from "graphql-tag";

export const statsProjectFindMany = gql`
query statsProject($input: StatsProjectSearchInput!) {
  statsProject(input: $input) {
    edges {
        cursor
        node {
            id
            periodStart
            periodEnd
            periodType
            directories
            apis
            notes
            organizations
            projects
            routines
            smartContracts
            standards
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

