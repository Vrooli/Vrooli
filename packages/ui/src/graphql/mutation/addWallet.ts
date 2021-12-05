// First call in handshake for logging in with wallet.
// Name is a bit of a misnomer, as it is called even when you've logged in with the same wallet before
import { gql } from 'graphql-tag';

export const addWalletMutation = gql`
    mutation addWallet(
        $publicAddress: String!
        $nonceDescription: String
    ) {
    addWallet(
        publicAddress: $publicAddress
        nonceDescription: $nonceDescription
    )
}
`