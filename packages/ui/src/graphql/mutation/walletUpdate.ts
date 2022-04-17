import { gql } from 'graphql-tag';
import { walletFields } from 'graphql/fragment';

export const walletUpdateMutation = gql`
    ${walletFields}
    mutation walletUpdate($input: WalletUpdateInput!) {
        walletUpdate(input: $input) {
            ...walletFields
        }
}
`