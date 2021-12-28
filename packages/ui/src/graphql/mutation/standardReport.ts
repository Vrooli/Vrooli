import { gql } from 'graphql-tag';

export const standardReportMutation = gql`
    mutation standardReport($input: ReportInput!) {
        standardReport(input: $input) {
            success
        }
    }
`