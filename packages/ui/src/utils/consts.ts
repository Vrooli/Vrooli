/* eslint-disable @typescript-eslint/no-redeclare */
import { COOKIE, ValueOf } from '@local/shared';

export const Pubs = {
    ...COOKIE,
    Loading: "loading",
    AlertDialog: "alertDialog",
    Snack: "snack",
    BurgerMenuOpen: "burgerMenuOpen",
    ArrowMenuOpen: "arrowMenuOpen",
    Theme: "theme",
}
export type Pubs = ValueOf<typeof Pubs>;

export const Forms = {
    ForgotPassword: 'forgot-password',
    LogIn: 'logIn',
    Profile: 'profile',
    ResetPassword: 'reset-password',
    SignUp: 'signUp',
}
export type Forms = ValueOf<typeof Forms>;

/**
 * Converts GraphQL sort values to User-Friendly labels
 */
export const SortValueToLabelMap = {
    'AlphabeticalAsc': 'Z-A',
    'AlphabeticalDesc': 'A-Z',
    'CommentsDesc': 'Most Comments',
    'StarsDesc': 'Most Stars',
    'ForksDesc': 'Most Forks',
}
export type SortValueToLabelMap = ValueOf<typeof SortValueToLabelMap>;