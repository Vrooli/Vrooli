import { gql } from 'graphql-tag';

export const walletRemoveMutation = gql`
    mutation walletRemove($input: DeleteOneInput!) {
        walletRemove(input: $input) {
            success
        }
    }
`