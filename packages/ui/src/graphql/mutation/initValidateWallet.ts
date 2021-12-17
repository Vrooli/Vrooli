// First call in handshake for logging in with wallet.
import { gql } from 'graphql-tag';

export const initValidateWalletMutation = gql`
    mutation initValidateWallet($input: InitValidateWalletInput!) {
    initValidateWallet(input: $input)
}
`