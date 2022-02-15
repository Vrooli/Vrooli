import { gql } from 'graphql-tag';
import { reportFields } from 'graphql/fragment';

export const reportCreateMutation = gql`
    ${reportFields}
    mutation reportCreate($input: ReportCreateInput!) {
        reportCreate(input: $input) {
            ...reportFields
        }
    }
`