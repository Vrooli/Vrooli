// Second call in handshake for logging in with wallet (must call walletInit first to get nonce)
import { gql } from 'graphql-tag';
import { sessionFields, walletFields } from 'graphql/fragment';

export const walletCompleteMutation = gql`
    ${sessionFields}
    ${walletFields}
    mutation walletComplete($input: WalletCompleteInput!) {
        walletComplete(input: $input) {
            firstLogIn
            session {
                ...sessionFields
            }
            wallet {
                ...walletFields
            }
        }
    }
`