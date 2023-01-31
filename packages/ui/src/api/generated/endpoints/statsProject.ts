import gql from 'graphql-tag';

export const statsProjectFindMany = gql`
query statsProject($input: StatsProjectSearchInput!) {
  statsProject(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            periodStart
            periodEnd
            periodType
            directories
            notes
            routines
            smartContracts
            standards
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

