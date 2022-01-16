import { gql } from 'graphql-tag';
import { reportFields } from 'graphql/fragment';

export const reportUpdateMutation = gql`
    ${reportFields}
    mutation reportUpdate($input: ReportInput!) {
        reportUpdate(input: $input) {
            ...reportFields
        }
    }
`