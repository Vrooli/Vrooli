import { Api, ApiVersion, Note, NoteVersion, Organization, Project, ProjectVersion, Routine, RoutineVersion, Session, SmartContract, SmartContractVersion, Standard, StandardVersion, User } from "@shared/consts";

export type SelectOrCreateObjectType = 'Api' | 'ApiVersion' | 'Note' | 'NoteVersion' | 'Organization' | 'Project' | 'ProjectVersion' | 'Routine' | 'RoutineVersion' | 'SmartContract' | 'SmartContractVersion' | 'Standard' | 'StandardVersion' | 'User';
export type SelectOrCreateObject = Api |
    ApiVersion |
    Note |
    NoteVersion |
    Organization |
    Project |
    ProjectVersion |
    Routine |
    RoutineVersion |
    SmartContract |
    SmartContractVersion |
    Standard |
    StandardVersion |
    User;

export interface SelectOrCreateDialogProps<T extends SelectOrCreateObject> {
    handleAdd: (item: T) => any;
    handleClose: () => any;
    help?: string;
    isOpen: boolean;
    objectType: SelectOrCreateObjectType;
    session: Session;
    where?: { [key: string]: any };
    zIndex: number;
}

export interface SubroutineSelectOrCreateDialogProps {
    handleAdd: (nodeId: string, subroutine: RoutineVersion) => any;
    handleClose: () => any;
    isOpen: boolean;
    owner: { type: 'Organization' | 'User', id: string } | null;
    nodeId: string;
    routineVersionId: string | null | undefined;
    session: Session;
    zIndex: number;
}