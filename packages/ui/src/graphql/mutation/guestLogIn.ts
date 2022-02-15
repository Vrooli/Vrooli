import { gql } from 'graphql-tag';
import { sessionFields } from 'graphql/fragment';

export const guestLogInMutation = gql`
    ${sessionFields}
    mutation guestLogIn {
        guestLogIn {
            ...sessionFields
        }
    }
`