import gql from "graphql-tag";
export const authWalletInit = gql `
mutation walletInit($input: WalletInitInput!) {
  walletInit(input: $input)
}`;
//# sourceMappingURL=auth_walletInit.js.map