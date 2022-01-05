import { 
    Organization,
    Project,
    Resource,
    RoutineShallow,
    Standard,
    User 
} from 'types';

export interface ActorCardProps {
    data: User;
    onClick?: (username: string) => void;
}

export interface OrganizationCardProps {
    data: Organization;
    onClick?: (name: string) => void;
}

export interface ProjectCardProps {
    data: Project;
    onClick?: (name: string) => void;
}

export interface ResourceCardProps {
    data: Resource
}

export interface RoutineCardProps {
    data: RoutineShallow;
    onClick?: (name: string) => void;
}

export interface StandardCardProps {
    data: Standard;
    onClick?: (name: string) => void;
}

export interface StatCardProps {
    data: any;
    index: number;
}