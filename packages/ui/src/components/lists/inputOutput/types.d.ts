import { Session } from "types";
import { InputCreate, OutputCreate } from "utils";

export interface InputOutputContainerProps {
    handleUpdate: (updatedList: (InputCreate | OutputCreate)[]) => void;
    isEditing: boolean;
    isInput: boolean;
    language: string;
    list: (InputCreate | OutputCreate)[];
    session: Session;
    zIndex: number;
}

export interface InputOutputListItemProps {
    index: number;
    isEditing: boolean;
    isInput: boolean;
    isOpen: boolean;
    item: InputCreate | OutputCreate;
    handleOpen: (index: number) => void;
    handleClose: (index: number) => void;
    handleDelete: (index: number) => void;
    handleUpdate: (index: number, updatedItem: InputCreate | OutputCreate) => void;
    language: string;
    session: Session;
    zIndex: number;
}