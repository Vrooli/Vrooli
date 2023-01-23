import { sessionPartial } from '../partial';
import { successPartial } from '../partial/success';
import { walletCompletePartial } from '../partial/walletComplete';
import { toMutation } from '../utils';

export const authEndpoint = {
    emailLogIn: toMutation('emailLogIn', 'EmailLogInInput', sessionPartial, 'full'),
    emailSignUp: toMutation('emailSignUp', 'EmailSignUpInput', sessionPartial, 'full'),
    emailRequestPasswordChange: toMutation('emailRequestPasswordChange', 'EmailRequestPasswordChangeInput', successPartial, 'full'),
    emailResetPassword: toMutation('emailResetPassword', 'EmailResetPasswordInput', sessionPartial, 'full'),
    guestLogIn: toMutation('guestLogIn', null, sessionPartial, 'full'),
    logOut: toMutation('logOut', 'LogOutInput', sessionPartial, 'full'),
    validateSession: toMutation('validateSession', 'ValidateSessionInput', sessionPartial, 'full'),
    switchCurrentAccount: toMutation('switchCurrentAccount', 'SwitchCurrentAccountInput', sessionPartial, 'full'),
    walletInit: toMutation('walletInit', 'WalletInitInput'),
    walletComplete: toMutation('walletComplete', 'WalletCompleteInput', walletCompletePartial, 'full'),
}