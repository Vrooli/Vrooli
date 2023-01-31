import gql from 'graphql-tag';

export const findMany = gql`
query statsStandard($input: StatsStandardSearchInput!) {
  statsStandard(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            periodStart
            periodEnd
            periodType
            linksToInputs
            linksToOutputs
            timesUsedInCompletedRoutines
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

