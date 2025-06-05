import { type Email, type Phone, type PushDevice, type Wallet } from "@vrooli/shared";

export interface EmailListProps {
    handleUpdate: (emails: Email[]) => unknown;
    numOtherVerified: number;
    list: Email[];
}

export interface EmailListItemProps {
    handleDelete: (email: Email) => unknown;
    handleVerify: (email: Email) => unknown;
    index: number;
    data: Email;
}

export interface PhoneListProps {
    handleUpdate: (Phones: Phone[]) => unknown;
    numOtherVerified: number;
    list: Phone[];
}

export interface PhoneListItemProps {
    data: Phone;
    handleDelete: (phone: Phone) => unknown;
    handleUpdate: (index: number, phone: Phone) => unknown;
    index: number;
}

export interface PushListProps {
    handleUpdate: (devices: PushDevice[]) => unknown;
    list: PushDevice[];
}

export interface PushListItemProps {
    handleDelete: (device: PushDevice) => unknown;
    handleTestPush: (device: PushDevice) => unknown;
    handleUpdate: (index: number, device: PushDevice) => unknown;
    index: number;
    data: PushDevice;
}

export interface WalletListProps {
    handleUpdate: (wallets: Wallet[]) => unknown;
    numOtherVerified: number;
    list: Wallet[];
}

export interface WalletListItemProps {
    handleDelete: (wallet: Wallet) => unknown;
    handleUpdate: (index: number, wallet: Wallet) => unknown;
    handleVerify: (wallet: Wallet) => unknown;
    index: number;
    data: Wallet;
}
