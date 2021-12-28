import { gql } from 'graphql-tag';

export const routineDeleteOneMutation = gql`
    mutation routineDeleteOne($input: DeleteOneInput!) {
        routineDeleteOne(input: $input) {
            success
        }
    }
`