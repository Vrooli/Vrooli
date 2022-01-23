import { Organization, Project, Resource, Routine, Session, Standard, User } from 'types';

export interface ObjectListItemProps {
    session: Session;
    isOwn: boolean;
    onClick?: (data: any) => void; // Full data passed back, to display while more details load
}

export interface ActorListItemProps extends ObjectListItemProps {
    data: Partial<User>;
}

export interface OrganizationListItemProps extends ObjectListItemProps {
    data: Partial<Organization>;
}

export interface ProjectListItemProps extends ObjectListItemProps {
    data: Partial<Project>;
}

export interface RoutineListItemProps extends ObjectListItemProps {
    data: Partial<Routine>;
}

export interface StandardListItemProps extends ObjectListItemProps {
    data: Partial<Standard>;
}

export interface SortMenuProps {
    sortOptions: any[];
    anchorEl: HTMLElement | null;
    onClose: (label?: string, value?: string) => void;
}

export interface TimeMenuProps {
    anchorEl: HTMLElement | null;
    onClose: (label?: string, after?: Date | null, before?: Date | null) => void;
}

export interface DateRangeMenuProps {
    anchorEl: HTMLElement | null;
    onClose: () => void;
    onSubmit: (after?: Date | null, before?: Date | null) => void;
}

export interface FeedListProps {
    title?: string;
    onClick: () => void;
    children: JSX.Element[];
}

export interface ResourceListProps {
    title?: string;
    canEdit?: boolean;
}

export interface StatsListProps {
    data: Array<any>;
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

export interface UpvoteDownvoteProps {
    session: Session;
    score?: number; // Net score - can be negative
    isUpvoted?: boolean | null; // If not passed, then there is neither an upvote nor a downvote
    onVote: (event: any, isUpvote: boolean | null) => void;
}

export interface StarButtonProps {
    session: Session;
    isStar?: boolean | null; // Defaults to false
    stars?: number | null; // Defaults to 0
    onStar: (event: any, isStar: boolean) => void;
}