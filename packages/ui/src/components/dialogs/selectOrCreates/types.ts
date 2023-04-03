import { RoutineVersion } from "@shared/consts";

export interface SelectOrCreateDialogProps<T extends SelectOrCreateObject> {
    handleAdd: (item: T) => any;
    handleClose: () => any;
    help?: string;
    isOpen: boolean;
    objectType: SelectOrCreateObjectType;
    where?: { [key: string]: any };
    zIndex: number;
}

export interface SubroutineSelectOrCreateDialogProps {
    handleAdd: (nodeId: string, subroutine: RoutineVersion) => any;
    handleClose: () => any;
    isOpen: boolean;
    nodeId: string;
    routineVersionId: string | null | undefined;
    zIndex: number;
}