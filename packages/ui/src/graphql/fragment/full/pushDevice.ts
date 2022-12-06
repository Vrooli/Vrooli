import { gql } from 'graphql-tag';

export const pushDeviceFields = gql`
    fragment pushDeviceFields on PushDevice {
        id
        expires
        name
    }
`