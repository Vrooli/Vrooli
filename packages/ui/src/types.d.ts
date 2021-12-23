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
export type User = users_users;
export type Resource = any; //TODO

// Enable Nami integration
declare global {
    interface Window { cardano: any; }
}
window.cardano = window.cardano || {};