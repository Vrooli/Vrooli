import { gql } from 'graphql-tag';
import { pushDeviceFields } from 'graphql/fragment';

export const pushDeviceUpdateMutation = gql`
    ${pushDeviceFields}
    mutation pushDeviceUpdate($input: PushDeviceUpdateInput!) {
        pushDeviceUpdate(input: $input) {
            ...pushDeviceFields
        }
    }
`