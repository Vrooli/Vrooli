import { gql } from 'graphql-tag';
import { customerSessionFields } from 'graphql/fragment';

export const signUpMutation = gql`
    ${customerSessionFields}
    mutation signUp(
        $firstName: String!
        $lastName: String!
        $pronouns: String
        $email: String!
        $theme: String!
        $marketingEmails: Boolean!
        $password: String!
    ) {
    signUp(
        firstName: $firstName
        lastName: $lastName
        pronouns: $pronouns
        email: $email
        theme: $theme
        marketingEmails: $marketingEmails
        password: $password
    ) {
        ...customerSessionFields
    }
}
`