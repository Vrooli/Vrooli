import { Email } from "@local/consts";

export interface EmailListProps {
    handleUpdate: (emails: Email[]) => void;
    numVerifiedWallets: number;
    list: Email[];
}

export interface EmailListItemProps {
    handleDelete: (email: Email) => void;
    handleVerify: (email: Email) => void;
    index: number;
    data: Email;
}
