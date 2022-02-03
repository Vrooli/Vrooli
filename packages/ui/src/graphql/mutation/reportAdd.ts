import { gql } from 'graphql-tag';
import { reportFields } from 'graphql/fragment';

export const reportAddMutation = gql`
    ${reportFields}
    mutation reportAdd($input: ReportAddInput!) {
        reportAdd(input: $input) {
            ...reportFields
        }
    }
`