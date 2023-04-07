import { DraggableProvidedDraggableProps, DraggableProvidedDragHandleProps } from "react-beautiful-dnd";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";

export interface InputOutputContainerProps {
    handleUpdate: (updatedList: (RoutineVersionInputShape | RoutineVersionOutputShape)[]) => void;
    isEditing: boolean;
    isInput: boolean;
    language: string;
    list: (RoutineVersionInputShape | RoutineVersionOutputShape)[];
    zIndex: number;
}

export interface InputOutputListItemProps {
    dragProps: DraggableProvidedDraggableProps;
    dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
    index: number;
    isEditing: boolean;
    isInput: boolean;
    isOpen: boolean;
    item: RoutineVersionInputShape | RoutineVersionOutputShape;
    handleOpen: (index: number) => void;
    handleClose: (index: number) => void;
    handleDelete: (index: number) => void;
    handleUpdate: (index: number, updatedItem: RoutineVersionInputShape | RoutineVersionOutputShape) => void;
    language: string;
    zIndex: number;
}