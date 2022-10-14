import { gql } from 'graphql-tag';
import { sessionFields } from 'graphql/fragment';

export const logOutMutation = gql`
    ${sessionFields}
    mutation logOut($input: LogOutInput!) {
        logOut(input: $input) {
            ...sessionFields
        }
    }
`