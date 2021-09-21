import { gql } from 'graphql-tag';

export const joinWaitlistMutation = gql`
    mutation joinWaitlist(
        $email: String!
    ) {
    joinWaitlist(
        email: $email
    )
}
`