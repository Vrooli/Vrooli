import { PushDevice } from "types";

export interface PushListProps {
    handleUpdate: (devices: PushDevice[]) => void;
    list: PushDevice[];
}

export interface PushListItemProps {
    handleDelete: (device: PushDevice) => void;
    handleUpdate: (index: number, device: PushDevice) => void;
    index: number;
    data: PushDevice;
}