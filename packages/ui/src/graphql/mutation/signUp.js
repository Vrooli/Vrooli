import { gql } from 'graphql-tag';
import { customerSessionFields } from 'graphql/fragment';

export const signUpMutation = gql`
    ${customerSessionFields}
    mutation signUp(
        $firstName: String!
        $lastName: String!
        $pronouns: String
        $business: String!
        $email: String!
        $phone: String!
        $theme: String!
        $marketingEmails: Boolean!
        $password: String!
    ) {
    signUp(
        firstName: $firstName
        lastName: $lastName
        pronouns: $pronouns
        business: $business
        email: $email
        phone: $phone
        theme: $theme
        marketingEmails: $marketingEmails
        password: $password
    ) {
        ...customerSessionFields
    }
}
`