import { Wallet } from "@local/shared";

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
