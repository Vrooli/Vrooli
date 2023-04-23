import gql from "graphql-tag";
export const walletFindHandles = gql `
query findHandles($input: FindHandlesInput!) {
  findHandles(input: $input)
}`;
//# sourceMappingURL=wallet_findHandles.js.map