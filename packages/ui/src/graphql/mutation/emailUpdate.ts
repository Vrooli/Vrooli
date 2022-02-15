import { gql } from 'graphql-tag';
import { emailFields } from 'graphql/fragment';

export const emailUpdateMutation = gql`
    ${emailFields}
    mutation emailUpdate($input: EmailUpdateInput!) {
        emailUpdate(input: $input) {
            ...emailFields
        }
    }
`