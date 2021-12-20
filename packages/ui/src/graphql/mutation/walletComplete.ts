// Second call in handshake for logging in with wallet (must call walletInit first to get nonce)
import { gql } from 'graphql-tag';

export const walletCompleteMutation = gql`
    mutation walletComplete($input: WalletCompleteInput!) {
        walletComplete(input: $input)
}
`