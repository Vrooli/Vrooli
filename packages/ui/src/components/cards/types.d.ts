import { Resource } from 'types';

export interface ResourceCardProps {
    canEdit: boolean;
    data: Resource;
    index: number;
    onContextMenu: (target: React.MouseEvent['target'], index: number) => void;
    session: Session;
}

export interface StatCardProps {
    title?: string;
    data: any;
    index: number;
}