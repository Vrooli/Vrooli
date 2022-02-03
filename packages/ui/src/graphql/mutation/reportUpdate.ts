import { gql } from 'graphql-tag';
import { reportFields } from 'graphql/fragment';

export const reportUpdateMutation = gql`
    ${reportFields}
    mutation reportUpdate($input: ReportUpdateInput!) {
        reportUpdate(input: $input) {
            ...reportFields
        }
    }
`