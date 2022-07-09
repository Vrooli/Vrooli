import { gql } from 'graphql-tag';
import { profileFields } from 'graphql/fragment';

export const profileEmailUpdateMutation = gql`
    ${profileFields}
    mutation profileEmailUpdate($input: ProfileEmailUpdateInput!) {
        profileEmailUpdate(input: $input) {
            ...profileFields
        }
    }
`