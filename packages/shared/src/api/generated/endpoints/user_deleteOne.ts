import gql from "graphql-tag";

export const userDeleteOne = gql`
mutation userDeleteOne($input: UserDeleteInput!) {
  userDeleteOne(input: $input) {
    success
  }
}`;

