import gql from 'graphql-tag';

export const findMany = gql`
query statsSmartContract($input: StatsSmartContractSearchInput!) {
  statsSmartContract(input: $input) {
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

