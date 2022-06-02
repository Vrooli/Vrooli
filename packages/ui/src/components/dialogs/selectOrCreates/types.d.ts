import { Organization, Routine, Standard } from "types";

export interface OrganizationSelectOrCreateDialogProps {
    handleAdd: (organization: Organization) => any;
    handleClose: () => any;
    isOpen: boolean;
    session: Session;
    zIndex: number;
}

export interface StandardSelectOrCreateDialogProps {
    handleAdd: (standard: Standard) => any;
    handleClose: () => any;
    isOpen: boolean;
    session: Session;
    zIndex: number;
}

export interface SubroutineSelectOrCreateDialogProps {
    handleAdd: (nodeId: string, subroutine: Routine) => any;
    handleClose: () => any;
    isOpen: boolean;
    nodeId: string;
    routineId: string;
    session: Session;
    zIndex: number;
}