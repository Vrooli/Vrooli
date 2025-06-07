import { type ModelType } from "@vrooli/shared";
import { type QueryAction } from "./types.js";

/**
 * Represents a node in a hierarchical tree structure built from a GraphQL mutation input.
 * 
 * The `InputNode` class is designed to facilitate the optimization of various operations
 * performed before and after creating, updating, deleting, connecting, and disconnecting objects.
 * It is used in the construction of a tree that represents the relationships and dependencies
 * between different entities in a mutation request.
 *
 * Each `InputNode` instance holds information about a specific entity, including its type,
 * identifier, and the action to be performed on it (e.g., create, update, delete, connect, disconnect).
 * The class also maintains references to child nodes (representing related entities) and a parent node,
 * enabling the traversal and manipulation of the hierarchical structure.
 *
 * In cases where operations like "Connect" or "Disconnect" are performed on non-array relations
 * (such as one-to-one or many-to-one relationships) without explicit knowledge of the object's ID,
 * `InputNode` can handle placeholder IDs. These placeholders encode necessary information to query
 * the database for the actual IDs, ensuring the correct linking of entities.
 */
export class InputNode {
    __typename: `${ModelType}`;
    id: string;
    action: QueryAction;
    children: InputNode[];
    parent: InputNode | null;

    constructor(__typename: `${ModelType}`, id: string, action: QueryAction) {
        this.__typename = __typename;
        this.id = id;
        this.action = action;
        this.children = [];
        this.parent = null;
    }
}
