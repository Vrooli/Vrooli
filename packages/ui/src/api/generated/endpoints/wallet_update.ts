import gql from 'graphql-tag';

export const walletUpdate = gql`
mutation walletUpdate($input: WalletUpdateInput!) {
  walletUpdate(input: $input) {
    id
    handles {
        id
        handle
    }
    name
    publicAddress
    stakingAddress
    verified
  }
}`;

