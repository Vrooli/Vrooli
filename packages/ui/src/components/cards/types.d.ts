import { Resource } from 'types';

export interface ResourceCardProps {
    canEdit: boolean;
    data: Resource;
    handleEdit: (index: number) => void;
    handleDelete: (index: number) => void;
    index: number;
    onRightClick: (target: React.MouseEvent['target'], index: number) => void;
    session: Session;
}

export interface StatCardProps {
    title?: string;
    data: any;
    index: number;
}