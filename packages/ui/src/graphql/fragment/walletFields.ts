import { gql } from 'graphql-tag';

export const walletFields = gql`
    fragment walletFields on Wallet {
        id
        name
        publicAddress
        stakingAddress
        handles
        verified
    }
`