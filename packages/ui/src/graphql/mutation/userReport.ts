import { gql } from 'graphql-tag';

export const userReportMutation = gql`
    mutation userReport($input: ReportInput!) {
        userReport(input: $input) {
            success
        }
    }
`