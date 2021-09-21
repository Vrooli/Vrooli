import { gql } from 'graphql-tag';

export const joinWaitlistMutation = gql`
    mutation joinWaitlist(
        $firstName: String!
        $lastName: String!
        $pronouns: String!
        $email: String!
    ) {
    joinWaitlist(
        firstName: $firstName
        lastName: $lastName
        pronouns: $pronouns
        email: $email
    )
}
`