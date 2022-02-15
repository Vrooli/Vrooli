import { gql } from 'graphql-tag';
import { profileFields } from 'graphql/fragment';

export const profileUpdateMutation = gql`
    ${profileFields}
    mutation profileUpdate($input: ProfileUpdateInput!) {
        profileUpdate(input: $input) {
            ...profileFields
        }
    }
`