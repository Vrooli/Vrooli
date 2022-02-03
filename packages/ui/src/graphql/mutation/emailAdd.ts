import { gql } from 'graphql-tag';
import { emailFields } from 'graphql/fragment';

export const emailAddMutation = gql`
    ${emailFields}
    mutation emailAdd($input: EmailAddInput!) {
        emailAdd(input: $input) {
            ...emailFields
        }
    }
`