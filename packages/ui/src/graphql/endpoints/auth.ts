import { sessionFields, walletFields } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const authEndpoint = {
    emailLogIn: toMutation('emailLogIn', 'EmailLogInInput', [sessionFields], `...sessionFields`),
    emailSignUp: toMutation('emailSignUp', 'EmailSignUpInput', [sessionFields], `...sessionFields`),
    emailRequestPasswordChange: toMutation('emailRequestPasswordChange', 'EmailRequestPasswordChangeInput', [], `success`),
    emailResetPassword: toMutation('emailResetPassword', 'EmailResetPasswordInput', [sessionFields], `...sessionFields`),
    guestLogIn: toMutation('guestLogIn', null, [sessionFields], `...sessionFields`),
    logOut: toMutation('logOut', 'LogOutInput', [sessionFields], `...sessionFields`),
    validateSession: toMutation('validateSession', 'ValidateSessionInput', [sessionFields], `...sessionFields`),
    switchCurrentAccount: toMutation('switchCurrentAccount', 'SwitchCurrentAccountInput', [sessionFields], `...sessionFields`),
    walletInit: toMutation('walletInit', 'WalletInitInput', [], null),
    walletComplete: toMutation('walletComplete', 'WalletCompleteInput', [sessionFields, walletFields], `
        firstLogIn
        session {
            ...sessionFields
        }
        wallet {
            ...walletFields
        }
    `),
}