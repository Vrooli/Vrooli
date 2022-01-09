import { User } from 'types';

export interface ActorListItemProps {
    data: User;
    isStarred: boolean;
    isOwn: boolean;
    onClick: (id: string) => void;
    onStarClick: (id: string, removing: boolean) => void;
}

export interface FeedListProps<DataType> {
    title?: string;
    onClick: (pressedId?: string) => void;
    children: JSX.Element[];
}

export interface ResourceListProps {
    title?: string;
}

export interface StatsListProps {
    data: Array<any>;
}