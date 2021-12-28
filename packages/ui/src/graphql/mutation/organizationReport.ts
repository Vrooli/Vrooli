import { gql } from 'graphql-tag';

export const organizationReportMutation = gql`
    mutation organizationReport($input: ReportInput!) {
        organizationReport(input: $input) {
            success
        }
    }
`