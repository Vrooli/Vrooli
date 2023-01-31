import gql from 'graphql-tag';

export const findMany = gql`
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

