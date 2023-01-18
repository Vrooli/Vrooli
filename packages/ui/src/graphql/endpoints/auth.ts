import { sessionFields, walletFields } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const authEndpoint = {
    emailLogIn: toMutation('emailLogIn', 'EmailLogInInput', sessionFields[1]),
    emailSignUp: toMutation('emailSignUp', 'EmailSignUpInput', sessionFields[1]),
    emailRequestPasswordChange: toMutation('emailRequestPasswordChange', 'EmailRequestPasswordChangeInput', `{ success }`),
    emailResetPassword: toMutation('emailResetPassword', 'EmailResetPasswordInput', sessionFields[1]),
    guestLogIn: toMutation('guestLogIn', null, sessionFields[1]),
    logOut: toMutation('logOut', 'LogOutInput', sessionFields[1]),
    validateSession: toMutation('validateSession', 'ValidateSessionInput', sessionFields[1]),
    switchCurrentAccount: toMutation('switchCurrentAccount', 'SwitchCurrentAccountInput', sessionFields[1]),
    walletInit: toMutation('walletInit', 'WalletInitInput', null),
    walletComplete: toMutation('walletComplete', 'WalletCompleteInput', `{
        firstLogIn
        session {
            ...walletComplete0
        }
        wallet {
            ...walletComplete1
        }
    }`, [sessionFields, walletFields]),
}