import gql from 'graphql-tag';

export const statsApiFindMany = gql`
query statsApi($input: StatsApiSearchInput!) {
  statsApi(input: $input) {
    edges {
        cursor
        node {
            id
            periodStart
            periodEnd
            periodType
            calls
            routineVersions
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

