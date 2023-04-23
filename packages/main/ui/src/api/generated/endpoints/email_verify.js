import gql from "graphql-tag";
export const emailVerify = gql `
mutation sendVerificationEmail($input: SendVerificationEmailInput!) {
  sendVerificationEmail(input: $input) {
    success
  }
}`;
//# sourceMappingURL=email_verify.js.map