// Define common props
import { customers_customers } from 'graphql/generated/customers';

// Top-level props that can be passed into any routed component
export type UserRoles = string[] | null;
export type SessionChecked = boolean;
export type OnSessionUpdate = any;
export interface CommonProps {
    userRoles: UserRoles;
    sessionChecked: SessionChecked;
    onSessionUpdate: OnSessionUpdate;
}

// Rename auto-generated query objects
export type Customer = customers_customers;
export type Resource = any; //TODO

// Enable Nami integration
declare global {
    interface Window { cardano: any; }
}
window.cardano = window.cardano || {};