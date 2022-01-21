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
    NodeDrag: "NodeDrag",
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

export const DragTypes = {
    Node: 'node',
}
export type DragTypes = ValueOf<typeof DragTypes>;