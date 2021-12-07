import { gql } from 'graphql-tag';
import { userSessionFields } from 'graphql/fragment';

export const signUpMutation = gql`
    ${userSessionFields}
    mutation signUp(
        $username: String!
        $email: String!
        $theme: String!
        $marketingEmails: Boolean!
        $password: String!
    ) {
    signUp(
        username: $username
        email: $email
        theme: $theme
        marketingEmails: $marketingEmails
        password: $password
    ) {
        ...userSessionFields
    }
}
`