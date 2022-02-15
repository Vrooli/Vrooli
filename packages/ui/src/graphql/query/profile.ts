import { gql } from 'graphql-tag';
import { profileFields } from 'graphql/fragment';

export const profileQuery = gql`
    ${profileFields}
    query profile {
        profile {
            ...profileFields
        }
    }
`