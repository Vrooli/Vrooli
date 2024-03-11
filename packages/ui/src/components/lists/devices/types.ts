import { Email, PushDevice, Wallet } from "@local/shared";

export interface EmailListProps {
    handleUpdate: (emails: Email[]) => unknown;
    numVerifiedWallets: number;
    list: Email[];
}

export interface EmailListItemProps {
    handleDelete: (email: Email) => unknown;
    handleVerify: (email: Email) => unknown;
    index: number;
    data: Email;
}

export interface PushListProps {
    handleUpdate: (devices: PushDevice[]) => unknown;
    list: PushDevice[];
}

export interface PushListItemProps {
    handleDelete: (device: PushDevice) => unknown;
    handleUpdate: (index: number, device: PushDevice) => unknown;
    index: number;
    data: PushDevice;
}

export interface WalletListProps {
    handleUpdate: (wallets: Wallet[]) => unknown;
    numVerifiedEmails: number;
    list: Wallet[];
}

export interface WalletListItemProps {
    handleDelete: (wallet: Wallet) => unknown;
    handleUpdate: (index: number, wallet: Wallet) => unknown;
    handleVerify: (wallet: Wallet) => unknown;
    index: number;
    data: Wallet;
}
