import { Wallet } from "@local/shared";

export interface WalletListProps {
    handleUpdate: (wallets: Wallet[]) => void;
    numVerifiedEmails: number;
    list: Wallet[];
}

export interface WalletListItemProps {
    handleDelete: (wallet: Wallet) => void;
    handleUpdate: (index: number, wallet: Wallet) => void;
    handleVerify: (wallet: Wallet) => void;
    index: number;
    data: Wallet;
}