import gql from 'graphql-tag';

export const findHandles = gql`
query findHandles($input: FindHandlesInput!) {
  findHandles(input: $input)
}`;

export const update = gql`
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

