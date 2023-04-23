import gql from "graphql-tag";
export const statsSmartContractFindMany = gql `
query statsSmartContract($input: StatsSmartContractSearchInput!) {
  statsSmartContract(input: $input) {
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
//# sourceMappingURL=statsSmartContract_findMany.js.map