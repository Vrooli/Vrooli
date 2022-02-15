import { gql } from 'graphql-tag';
import { sessionFields } from 'graphql/fragment';

export const emailSignUpMutation = gql`
    ${sessionFields}
    mutation emailSignUp($input: EmailSignUpInput!) {
        emailSignUp(input: $input) {
            ...sessionFields
        }
    }
`