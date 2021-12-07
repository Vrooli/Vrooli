import { gql } from 'graphql-tag';
import { emailFields } from './emailFields';

export const userContactFields = gql`
    ${emailFields}
    fragment userContactFields on User {
        id
        username
        emails {
            ...emailFields
        }
    }
`