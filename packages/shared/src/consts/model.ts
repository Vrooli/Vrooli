import { ValueOf } from '.';

export const AccountStatus = {
    DELETED: 'Deleted',
    UNLOCKED: 'Unlocked',
    SOFT_LOCKED: 'SoftLock',
    HARD_LOCKED: 'HardLock'
}
export type AccountStatus = ValueOf<typeof AccountStatus>;

export const ROLES = {
    Guest: 'Guest',
    Customer: 'Customer',
}
export type ROLES = ValueOf<typeof ROLES>;
