import { gql } from 'graphql-tag';

export const projectReportMutation = gql`
    mutation projectReport($input: ReportInput!) {
        projectReport(input: $input) {
            success
        }
    }
`