import { RoutineInput, RoutineInputList, RoutineOutput, RoutineOutputList } from "types";

export interface InputOutputContainerProps {
    handleUpdate: (updatedList: RoutineInputList | RoutineOutputList) => void;
    isEditing: boolean;
    isInput: boolean;
    list: RoutineInputList | RoutineOutputList;
    session: Session;
}

export interface InputOutputListItemProps {
    index: number;
    isEditing: boolean;
    isInput: boolean;
    isOpen: boolean;
    item: RoutineInput | RoutineOutput;
    handleOpen: (index: number) => void;
    handleClose: (index: number) => void;
    handleDelete: (index: number) => void;
    handleUpdate: (index: number, updatedItem: RoutineInput | RoutineOutput) => void;
    handleOpenStandardSelect: (index: number) => void;
    handleRemoveStandard: (index: number) => void;
}