import gql from "graphql-tag";
export const authEmailRequestPasswordChange = gql `
mutation emailRequestPasswordChange($input: EmailRequestPasswordChangeInput!) {
  emailRequestPasswordChange(input: $input) {
    success
  }
}`;
//# sourceMappingURL=auth_emailRequestPasswordChange.js.map