// Second call in handshake for logging in with wallet (must call walletInit first to get nonce)
import { gql } from 'graphql-tag';
import { sessionFields } from 'graphql/fragment';

export const walletCompleteMutation = gql`
    ${sessionFields}
    mutation walletComplete($input: WalletCompleteInput!) {
        walletComplete(input: $input) {
            ...sessionFields
        }
    }
`