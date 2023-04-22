import gql from "graphql-tag";

export const pushDeviceCreate = gql`
mutation pushDeviceCreate($input: PushDeviceCreateInput!) {
  pushDeviceCreate(input: $input) {
    id
    expires
    name
  }
}`;

