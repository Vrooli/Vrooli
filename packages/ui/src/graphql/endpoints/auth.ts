import { sessionFields } from 'graphql/fragment';
import { toGql } from 'graphql/utils';

export const authEndpoint = {
    emailLogIn: toGql('mutation', 'emailLogIn', 'EmailLogInInput', [sessionFields], `...sessionFields`),
    emailSignUp: toGql('mutation', 'emailSignUp', 'EmailSignUpInput', [sessionFields], `...sessionFields`),
    emailRequestPasswordChange: toGql('mutation', 'emailRequestPasswordChange', 'EmailRequestPasswordChangeInput', [], `success`),
    emailResetPassword: toGql('mutation', 'emailResetPassword', 'EmailResetPasswordInput', [sessionFields], `...sessionFields`),
    guestLogIn: toGql('mutation', 'guestLogIn', null, [sessionFields], `...sessionFields`),
    logOut: toGql('mutation', 'logOut', 'LogOutInput', [sessionFields], `...sessionFields`),
    validateSession: toGql('mutation', 'validateSession', 'ValidateSessionInput', [sessionFields], `...sessionFields`),
    switchCurrentAccount: toGql('mutation', 'switchCurrentAccount', 'SwitchCurrentAccountInput', [sessionFields], `...sessionFields`),
    walletInit: toGql('mutation', 'walletInit', 'WalletInitInput', [], null),
    walletComplete: toGql('mutation', 'walletComplete', 'WalletCompleteInput', [], `
        firstLogIn
        session {
            ...sessionFields
        }
        wallet {
            ...walletFields
        }
    `),
}