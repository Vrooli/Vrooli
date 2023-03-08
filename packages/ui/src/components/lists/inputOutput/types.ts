import { Session } from "@shared/consts/src";
import { RoutineVersionInputShape, RoutineVersionOutputShape } from "utils";

export interface InputOutputContainerProps {
    handleUpdate: (updatedList: (RoutineVersionInputShape | RoutineVersionOutputShape)[]) => void;
    isEditing: boolean;
    isInput: boolean;
    language: string;
    list: (RoutineVersionInputShape | RoutineVersionOutputShape)[];
    session: Session | undefined;
    zIndex: number;
}

export interface InputOutputListItemProps {
    index: number;
    isEditing: boolean;
    isInput: boolean;
    isOpen: boolean;
    item: RoutineVersionInputShape | RoutineVersionOutputShape;
    handleOpen: (index: number) => void;
    handleClose: (index: number) => void;
    handleDelete: (index: number) => void;
    handleReorder: (index: number) => void;
    handleUpdate: (index: number, updatedItem: RoutineVersionInputShape | RoutineVersionOutputShape) => void;
    language: string;
    session: Session | undefined;
    zIndex: number;
}