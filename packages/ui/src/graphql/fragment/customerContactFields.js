import { gql } from 'graphql-tag';
import { emailFields } from './emailFields';

export const customerContactFields = gql`
    ${emailFields}
    fragment customerContactFields on Customer {
        id
        firstName
        lastName
        fullName
        pronouns
        emails {
            ...emailFields
        }
    }
`