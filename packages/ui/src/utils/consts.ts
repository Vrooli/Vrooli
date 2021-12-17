/* eslint-disable @typescript-eslint/no-redeclare */
import { COOKIE, ValueOf } from '@local/shared';

export const PUBS = {
    ...COOKIE,
    Loading: "loading",
    AlertDialog: "alertDialog",
    Snack: "snack",
    BurgerMenuOpen: "burgerMenuOpen",
    ArrowMenuOpen: "arrowMenuOpen",
    Theme: "theme",
}
export type PUBS = ValueOf<typeof PUBS>;

export const FORMS = {
    ForgotPassword: 'forgot-password',
    LogIn: 'logIn',
    Profile: 'profile',
    ResetPassword: 'reset-password',
    SignUp: 'signUp',
}
export type FORMS = ValueOf<typeof FORMS>;