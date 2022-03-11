export interface ResourceListHorizontalProps {
    title?: string;
    canEdit?: boolean;
    handleUpdate?: (updatedList: Resource[]) => void;
    list: ResourceList | null;
    session: Session;
}

export interface ResourceListVerticalProps {
    title?: string;
    canEdit?: boolean;
    handleUpdate?: (updatedList: Resource[]) => void;
    list: ResourceList | null;
    session: session
}

export interface ResourceListItemProps {
    session: Session;
    index: number;
    data: Resource;
    onClick?: (e: any, data: any) => void;
}

export interface ResourceListItemContextMenuProps {
    id: string;
    anchorEl: HTMLElement | null;
    resource: Resource | null;
    onClose: () => void;
    onAddBefore: (resource: Resource) => void;
    onAddAfter: (resource: Resource) => void;
    onEdit: (resource: Resource) => void;
    onDelete: (resource: Resource) => void;
    onMove: (resource: Resource) => void;
}