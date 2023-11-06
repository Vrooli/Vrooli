import { DraggableProvidedDraggableProps, DraggableProvidedDragHandleProps } from "react-beautiful-dnd";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";

export interface InputOutputContainerProps {
    handleUpdate: (updatedList: (RoutineVersionInputShape | RoutineVersionOutputShape)[]) => unknown;
    isEditing: boolean;
    isInput: boolean;
    language: string;
    list: (RoutineVersionInputShape | RoutineVersionOutputShape)[];
}

export interface InputOutputListItemProps {
    dragProps: DraggableProvidedDraggableProps;
    dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
    index: number;
    isEditing: boolean;
    isInput: boolean;
    isOpen: boolean;
    item: RoutineVersionInputShape | RoutineVersionOutputShape;
    handleOpen: (index: number) => unknown;
    handleClose: (index: number) => unknown;
    handleDelete: (index: number) => unknown;
    handleUpdate: (index: number, updatedItem: RoutineVersionInputShape | RoutineVersionOutputShape) => unknown;
    language: string;
}
