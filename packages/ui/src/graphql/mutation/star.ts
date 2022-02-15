import { gql } from 'graphql-tag';

export const starMutation = gql`
    mutation star($input: StarInput!) {
        star(input: $input) {
            success
        }
    }
`