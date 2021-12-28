import { gql } from 'graphql-tag';
import { emailFields } from 'graphql/fragment';

export const emailAddMutation = gql`
    ${emailFields}
    mutation emailAdd($input: EmailInput!) {
        emailAdd(input: $input) {
            ...emailFields
        }
    }
`