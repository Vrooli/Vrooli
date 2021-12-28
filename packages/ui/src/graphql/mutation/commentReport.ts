import { gql } from 'graphql-tag';

export const commentReportMutation = gql`
    mutation commentReport($input: ReportInput!) {
        commentReport(input: $input) {
            success
        }
    }
`