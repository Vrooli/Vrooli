import { gql } from 'graphql-tag';
import { emailFields } from './emailFields';
import { phoneFields } from './phoneFields';

export const customerContactFields = gql`
    ${emailFields}
    ${phoneFields}
    fragment customerContactFields on Customer {
        id
        firstName
        lastName
        fullName
        pronouns
        emails {
            ...emailFields
        }
        phones {
            ...phoneFields
        }
        business {
            id
            name
        }
    }
`