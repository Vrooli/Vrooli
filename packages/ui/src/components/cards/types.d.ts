import { Resource } from 'types';

export interface ResourceCardProps {
    canEdit: boolean;
    data: Resource;
    handleEdit: (index: number) => void;
    handleDelete: (index: number) => void;
    index: number;
    onRightClick: (ev: any, index: number) => void;
    session: Session;
}

export interface StatCardProps {
    title?: string;
    data: any;
    index: number;
}