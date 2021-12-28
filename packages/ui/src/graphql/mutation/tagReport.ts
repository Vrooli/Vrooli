import { gql } from 'graphql-tag';

export const tagReportMutation = gql`
    mutation tagReport($input: ReportInput!) {
        tagReport(input: $input) {
            success
        }
    }
`