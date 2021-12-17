// Second call in handshake for logging in with wallet (must call initValidateWallet first to get nonce)
import { gql } from 'graphql-tag';

export const completeValidateWalletMutation = gql`
    mutation completeValidateWallet($input: CompleteValidateWalletInput!) {
        completeValidateWallet(input: $input)
}
`