import { gql } from 'graphql-tag';

export const resourceReportMutation = gql`
    mutation resourceReport($input: ReportInput!) {
        resourceReport(input: $input) {
            success
        }
    }
`