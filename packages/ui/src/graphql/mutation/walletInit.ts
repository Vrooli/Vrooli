// First call in handshake for logging in with wallet.
import { gql } from 'graphql-tag';

export const walletInitMutation = gql`
    mutation walletInit($input: WalletInitInput!) {
        walletInit(input: $input)
}
`