import { gql } from 'graphql-tag';

export const profileFields = gql`
    fragment profileFields on Profile {
        id
        username
    }
`