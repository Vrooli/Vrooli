import { gql } from 'graphql-tag';

export const joinWaitlistMutation = gql`
    mutation joinWaitlist(
        $username: String!
        $email: String!
    ) {
    joinWaitlist(
        username: $username
        email: $email
    )
}
`