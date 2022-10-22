import { gql } from 'graphql-tag';
import { sessionFields } from 'graphql/fragment';

export const switchCurrentAccountMutation = gql`
    ${sessionFields}
    mutation switchCurrentAccount($input: SwitchCurrentAccountInput!) {
        switchCurrentAccount(input: $input) {
            ...sessionFields
        }
    }
`