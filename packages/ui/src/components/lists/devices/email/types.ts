import { Email } from "@local/shared";

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
