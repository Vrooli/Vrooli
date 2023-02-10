import gql from 'graphql-tag';

export const statsOrganizationFindMany = gql`
query statsOrganization($input: StatsOrganizationSearchInput!) {
  statsOrganization(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            periodStart
            periodEnd
            periodType
            apis
            members
            notes
            projects
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
