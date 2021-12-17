import { gql } from 'graphql-tag';
import { userSessionFields } from 'graphql/fragment';

export const logInMutation = gql`
    ${userSessionFields}
    mutation logIn($input: LogInInput!) {
    logIn(input: $input) {
        ...userSessionFields
    }
}
`