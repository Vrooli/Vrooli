import { gql } from 'graphql-tag';
import { sessionFields } from 'graphql/fragment';

export const emailLogInMutation = gql`
    ${sessionFields}
    mutation emailLogIn($input: EmailLogInInput!) {
        emailLogIn(input: $input) {
            ...sessionFields
        }
    }
`