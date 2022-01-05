// Defines common props
import { organization_organization } from 'graphql/generated/organization';
import { project_project } from 'graphql/generated/project';
import { resource_resource } from 'graphql/generated/resource';
import { routine_routine } from 'graphql/generated/routine';
import { routines_routines_edges_node } from 'graphql/generated/routines';
import { standard_standard } from 'graphql/generated/standard';
import { tag_tag } from 'graphql/generated/tag';
import { user_user } from 'graphql/generated/user';

// Top-level props that can be passed into any routed component
export type UserRoles = { title: string, description?: string }[] | null;
export type SessionChecked = boolean;
export type Session = {
    id?: string,
    theme?: string;
    roles?: UserRoles;
}
export type OnSessionUpdate = any;
export interface CommonProps {
    session?: Session;
    userRoles: UserRoles;
    sessionChecked: SessionChecked;
    onSessionUpdate: OnSessionUpdate;
}

// Rename auto-generated query objects
export type Comment = any; //TODO
export type Email = any; //TODO
export type Node = any; //TODO
export type Organization = organization_organization;
export type Project = project_project;
export type Report = any; //TODO
export type Resource = resource_resource;
export type RoutineDeep = routine_routine;
export type RoutineShallow = routines_routines_edges_node;
export type Standard = standard_standard;
export type Tag = tag_tag;
export type User = user_user;
export type Wallet = any; //TODO

// Enable Nami integration
declare global {
    interface Window { cardano: any; }
}
window.cardano = window.cardano || {};