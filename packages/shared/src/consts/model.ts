import { ValueOf } from '.';

export const AccountStatus = {
    DELETED: 'Deleted',
    UNLOCKED: 'Unlocked',
    SOFT_LOCKED: 'SoftLock',
    HARD_LOCKED: 'HardLock'
}
export type AccountStatus = ValueOf<typeof AccountStatus>;

export const THEME = {
    Light: 'light',
    Dark: 'dark'
}
export type THEME = ValueOf<typeof THEME>;

export const ROLES = {
    Customer: "Customer",
}
export type ROLES = ValueOf<typeof ROLES>;
