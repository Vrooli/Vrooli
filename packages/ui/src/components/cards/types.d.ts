export interface ActorCardProps {
    username?: string;
    onClick?: (username: string) => void;
}

export interface OrganizationCardProps {
    name?: string;
    onClick?: (name: string) => void;
}

export interface ProjectCardProps {
    name?: string;
    onClick?: (name: string) => void;
}

export interface ResourceCardProps {
    resource: Resource
}
export interface RoutineCardProps {
    name?: string;
    onClick?: (name: string) => void;
}

export interface StatCardProps {
    data: any;
    index: number;
}