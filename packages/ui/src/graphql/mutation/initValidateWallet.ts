// First call in handshake for logging in with wallet.
import { gql } from 'graphql-tag';

export const initValidateWalletMutation = gql`
    mutation initValidateWallet(
        $publicAddress: String!
        $nonceDescription: String
    ) {
    initValidateWallet(
        publicAddress: $publicAddress
        nonceDescription: $nonceDescription
    )
}
`