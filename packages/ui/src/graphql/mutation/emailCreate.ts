import { gql } from 'graphql-tag';
import { emailFields } from 'graphql/fragment';

export const emailCreateMutation = gql`
    ${emailFields}
    mutation emailCreate($input: EmailCreateInput!) {
        emailCreate(input: $input) {
            ...emailFields
        }
    }
`