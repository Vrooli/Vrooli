import { endpointsAuth } from "@local/shared";
import { auth_emailLogIn, auth_emailRequestPasswordChange, auth_emailResetPassword, auth_emailSignUp, auth_guestLogIn, auth_logOut, auth_switchCurrentAccount, auth_validateSession, auth_walletComplete, auth_walletInit } from "../generated";
import { AuthEndpoints } from "../logic/auth";
import { setupRoutes } from "./base";

export const AuthRest = setupRoutes([
    [endpointsAuth.emailLogin, AuthEndpoints.Mutation.emailLogIn, auth_emailLogIn],
    [endpointsAuth.emailSignup, AuthEndpoints.Mutation.emailSignUp, auth_emailSignUp],
    [endpointsAuth.emailRequestPasswordChange, AuthEndpoints.Mutation.emailRequestPasswordChange, auth_emailRequestPasswordChange],
    [endpointsAuth.emailResetPassword, AuthEndpoints.Mutation.emailResetPassword, auth_emailResetPassword],
    [endpointsAuth.emailLogin, AuthEndpoints.Mutation.guestLogIn, auth_guestLogIn],
    [endpointsAuth.logout, AuthEndpoints.Mutation.logOut, auth_logOut],
    [endpointsAuth.logoutAll, AuthEndpoints.Mutation.logOutAll, auth_logOut],
    [endpointsAuth.validateSession, AuthEndpoints.Mutation.validateSession, auth_validateSession],
    [endpointsAuth.switchCurrentAccount, AuthEndpoints.Mutation.switchCurrentAccount, auth_switchCurrentAccount],
    [endpointsAuth.walletInit, AuthEndpoints.Mutation.walletInit, auth_walletInit],
    [endpointsAuth.walletComplete, AuthEndpoints.Mutation.walletComplete, auth_walletComplete],
]);
