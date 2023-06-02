import { auth_emailLogIn, auth_emailRequestPasswordChange, auth_emailResetPassword, auth_emailSignUp, auth_guestLogIn, auth_logOut, auth_switchCurrentAccount, auth_validateSession, auth_walletComplete, auth_walletInit } from "@local/shared";
import { AuthEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const AuthRest = setupRoutes({
    "/auth/email/login": {
        post: [AuthEndpoints.Mutation.emailLogIn, auth_emailLogIn],
    },
    "/auth/email/signup": {
        post: [AuthEndpoints.Mutation.emailSignUp, auth_emailSignUp],
    },
    "/auth/email/requestPasswordChange": {
        post: [AuthEndpoints.Mutation.emailRequestPasswordChange, auth_emailRequestPasswordChange],
    },
    "/auth/email/resetPassword": {
        post: [AuthEndpoints.Mutation.emailResetPassword, auth_emailResetPassword],
    },
    "/auth/guest/login": {
        post: [AuthEndpoints.Mutation.guestLogIn, auth_guestLogIn],
    },
    "/auth/logout": {
        post: [AuthEndpoints.Mutation.logOut, auth_logOut],
    },
    "/auth/validateSession": {
        post: [AuthEndpoints.Mutation.validateSession, auth_validateSession],
    },
    "/auth/switchCurrentAccount": {
        post: [AuthEndpoints.Mutation.switchCurrentAccount, auth_switchCurrentAccount],
    },
    "/auth/wallet/init": {
        post: [AuthEndpoints.Mutation.walletInit, auth_walletInit],
    },
    "/auth/wallet/complete": {
        post: [AuthEndpoints.Mutation.walletComplete, auth_walletComplete],
    },
});
