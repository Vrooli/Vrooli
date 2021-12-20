import { Role } from "../schema/types";

// Relationships
export type RoleRelationshipNames = 'users'
// Structure of object without relationships
export type RoleFields = Omit<Role, RoleRelationshipNames>;