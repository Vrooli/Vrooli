// Second call in handshake for logging in with wallet (must call addWallet first to get nonce)
import { gql } from 'graphql-tag';

export const verifyWalletMutation = gql`
    mutation verifyWallet(
        $publicAddress: String!
        $signedMessage: String!
    ) {
    verifyWallet(
        publicAddress: $publicAddress
        signedMessage: $signedMessage
    )
}
`