import gql from 'graphql-tag';

export const statsApiFindMany = gql`
query statsApi($input: StatsApiSearchInput!) {
  statsApi(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            periodStart
            periodEnd
            periodType
            calls
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

