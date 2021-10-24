import { ValueOf } from '.';

export const AccountStatus = {
    DELETED: 'Deleted',
    UNLOCKED: 'Unlocked',
    SOFT_LOCKED: 'SoftLock',
    HARD_LOCKED: 'HardLock'
}
export type AccountStatus = ValueOf<typeof AccountStatus>;

export const IMAGE_EXTENSION = {
    Bmp: '.bmp',
    Gif: '.gif',
    Png: '.png',
    Jpg: '.jpg',
    Jpeg: '.jpeg',
    Heic: '.heic',
    Heif: '.heif',
    Ico: '.ico'
}
export type IMAGE_EXTENSION = ValueOf<typeof IMAGE_EXTENSION>;

// Possible image sizes stored, and their max size
export const IMAGE_SIZE = {
    XXS: 32,
    XS: 64,
    S: 128,
    M: 256,
    ML: 512,
    L: 1024,
    XL: 2048,
    XXL: 4096,
}
export type IMAGE_SIZE = ValueOf<typeof IMAGE_SIZE>;

export const ImageUse = {
    HERO: 'HERO',
    HOME: 'HOME',
    MISSION: 'MISSION',
    ABOUT: 'ABOUT',
}
export type ImageUse = ValueOf<typeof ImageUse>;

export const THEME = {
    Light: 'light',
    Dark: 'dark'
}
export type THEME = ValueOf<typeof THEME>;

export const ROLES = {
    Customer: "Customer",
    Owner: "Owner",
    Admin: "Admin",
}
export type ROLES = ValueOf<typeof ROLES>;
