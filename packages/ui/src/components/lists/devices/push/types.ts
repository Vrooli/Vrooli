import { PushDevice } from "@local/shared";

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
