// First call in handshake for logging in with wallet.
import { gql } from 'graphql-tag';

export const exportDataMutation = gql`
    mutation exportData {
        exportData
    }
`