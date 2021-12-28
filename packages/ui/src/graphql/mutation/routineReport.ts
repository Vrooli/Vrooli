import { gql } from 'graphql-tag';

export const routineReportMutation = gql`
    mutation routineReport($input: ReportInput!) {
        routineReport(input: $input) {
            success
        }
    }
`