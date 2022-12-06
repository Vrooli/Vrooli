import { gql } from 'graphql-tag';
import { pushDeviceFields } from 'graphql/fragment';

export const pushDeviceCreateMutation = gql`
    ${pushDeviceFields}
    mutation pushDeviceCreate($input: PushDeviceCreateInput!) {
        pushDeviceCreate(input: $input) {
            ...pushDeviceFields
        }
    }
`