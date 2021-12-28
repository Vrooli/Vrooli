// Defines common props
import { users_users } from 'graphql/generated/users';

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
export type Organization = any; //TODO
export type Project = any; //TODO
export type Report = any; //TODO
export type Resource = any; //TODO
export type Routine = any; //TODO
export type Standard = any; //TODO
export type Tag = any; //TODO
export type User = users_users;
export type Wallet = any; //TODO

// Enable Nami integration
declare global {
    interface Window { cardano: any; }
}
window.cardano = window.cardano || {};