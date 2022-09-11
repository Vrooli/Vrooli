import { Session } from "types";
import { InputShape, OutputShape } from "utils";

export interface InputOutputContainerProps {
    handleUpdate: (updatedList: (InputShape | OutputShape)[]) => void;
    isEditing: boolean;
    isInput: boolean;
    language: string;
    list: (InputShape | OutputShape)[];
    session: Session;
    zIndex: number;
}

export interface InputOutputListItemProps {
    index: number;
    isEditing: boolean;
    isInput: boolean;
    isOpen: boolean;
    item: InputShape | OutputShape;
    handleOpen: (index: number) => void;
    handleClose: (index: number) => void;
    handleDelete: (index: number) => void;
    handleReorder: (index: number) => void;
    handleUpdate: (index: number, updatedItem: InputShape | OutputShape) => void;
    language: string;
    session: Session;
    zIndex: number;
}