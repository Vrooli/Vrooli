import { gql } from 'graphql-tag';

export const verifyWaitlistMutation = gql`
    mutation verifyWaitlist(
        $confirmationCode: String!
    ) {
    verifyWaitlist(
        confirmationCode: $confirmationCode
    )
}
`