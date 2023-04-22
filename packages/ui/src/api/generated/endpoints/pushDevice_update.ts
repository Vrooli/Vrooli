import gql from "graphql-tag";

export const pushDeviceUpdate = gql`
mutation pushDeviceUpdate($input: PushDeviceUpdateInput!) {
  pushDeviceUpdate(input: $input) {
    id
    expires
    name
  }
}`;

