import { gql } from 'graphql-tag';

export const userDeleteOneMutation = gql`
    mutation userDeleteOne($input: UserDeleteInput!) {
        userDeleteOne(input: $input) {
            success
        }
    }
`