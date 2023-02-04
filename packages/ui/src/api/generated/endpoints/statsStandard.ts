import gql from 'graphql-tag';

export const statsStandardFindMany = gql`
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
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

