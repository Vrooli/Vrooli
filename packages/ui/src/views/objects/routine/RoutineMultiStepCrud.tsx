import { Box, Button, Divider, Grid, IconButton, styled, useTheme } from "@mui/material";
import { DUMMY_ID, addToArray, deleteArrayIndex, exists, getTranslation, shapeRoutineVersion, updateArray, uuid, type DefinedArrayElement, type RoutineVersion, type RoutineVersionShape, type Session } from "@vrooli/shared";
// eslint-disable-next-line import/extensions
// import Modeler from "bpmn-js/lib/Modeler";
import { BottomActionsGrid } from "../../../components/buttons/BottomActionsGrid.js";
import { LoadableButton } from "../../../components/buttons/LoadableButton.js";
import { type StatusMessageArray } from "../../../components/buttons/types.js";
// eslint-disable-next-line import/extensions
// import Canvas from "diagram-js/lib/core/Canvas";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useErrorPopover } from "../../../hooks/useErrorPopover.js";
import { IconCommon } from "../../../icons/Icons.js";
import { pagePaddingBottom } from "../../../styles.js";
import { type FormErrors, type FormProps } from "../../../types.js";
import { type RoutineMultiStepCrudProps } from "./types.js";

enum NodeType {
    Redirect = "Redirect",
    RoutineList = "RoutineList",
    End = "End",
}

export type NodeLink = any;//DefinedArrayElement<RoutineVersionShape["nodeLinks"]>;

type Node = any;//DefinedArrayElement<RoutineVersionShape["nodes"]>;
export type NodeEnd = any;//Node & { nodeType: NodeType.End, end: NonNullable<NodeShape["end"]> };
export type NodeLoop = Node;
export type NodeRedirect = Node & { nodeType: NodeType.Redirect };
export type NodeRoutineList = any;//Node & { nodeType: NodeType.RoutineList, routineList: NonNullable<NodeShape["routineList"]> };
export type NodeStart = any;//Node & { nodeType: NodeType.Start };
export type SomeNode = NodeEnd | NodeLoop | NodeRedirect | NodeRoutineList | NodeStart;

export type RoutineListItem = DefinedArrayElement<NodeRoutineList["routineList"]["items"]>;

type GraphElementOperation = "create" | "update" | "delete";
// type GraphOperation = Exclude<keyof typeof OperationManager, "prototype">;
export type NodeOperation = GraphElementOperation;
export type SubroutineOperation = GraphElementOperation;

export type LinkOperation = "AddNode" | "Branch" | "DeleteLink" | "EditLink"; // | "SetConditions";

const DEFAULT_SCALE_PERCENT = 100;
const SCALE_FACTOR = 0.1;

/**
 * Generates graph elements for a routine version, but doesn't add them to the routine 
 * or relevant parent.
 */
class GenerateManager {
    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    /**
     * Generates a new link object, but doesn't add it to the routine
     * 
     * @param fromId The ID of the node the link is coming from
     * @param toId The ID of the node the link is going to
     * @returns The new link object
     */
    static link(fromId: string, toId: string): NodeLink {
        return {
            __typename: "NodeLink",
            id: uuid(),
            from: { __typename: "Node", id: fromId },
            to: { __typename: "Node", id: toId },
        };
    }

    /**
     * Generates a new end node object, but doesn't add it to the routine
     * 
     * @param language The language to add the end node in
     */
    static nodeEnd(
        language: string,
    ): NodeEnd {
        return {
            __typename: "Node" as const,
            id: uuid(),
            nodeType: NodeType.End,
            rowIndex: 0, // Deprecated
            columnIndex: 0, // Deprecated
            end: {
                __typename: "NodeEnd" as const,
                id: uuid(),
                wasSuccessful: true,
            },
            translations: [{
                __typename: "NodeTranslation" as const,
                id: DUMMY_ID,
                language,
                name: "End",
                description: "",
            }],
        };
    }

    /**
     * Generates a new routine list node object, but doesn't add it to the routine
     * 
     * @param routine The routine version to add the node to
     * @param language The language to add the node in
     */
    static nodeRoutineList(
        routine: RoutineVersionShape,
        language: string,
    ): NodeRoutineList {
        return {
            // __typename: "Node" as const,
            // id: uuid(),
            // nodeType: NodeType.RoutineList,
            // rowIndex: 0, // Deprecated
            // columnIndex: 0, // Deprecated
            // routineList: {
            //     __typename: "NodeRoutineList" as const,
            //     id: uuid(),
            //     isOrdered: false,
            //     isOptional: false,
            //     items: [],
            // },
            // translations: [{
            //     __typename: "NodeTranslation" as const,
            //     id: uuid(),
            //     language,
            //     name: `Node ${(routine.nodes?.length ?? 0) - 1}`,
            //     description: "",
            // }],
        } as any;
    }

    /**
     * Generates a routine list item object, but doesn't add it to the routine list node
     * 
     * @param node The routine list node to add the item to
     * @param subroutine The subroutine to add
     * @param language The language to add the item in
     * @returns The new routine list item object
     */
    static routineListItem(
        node: NodeRoutineList,
        subroutine: RoutineVersion,
        language: string,
    ): RoutineListItem {
        return {
            __typename: "NodeRoutineListItem" as const,
            id: uuid(),
            index: node.routineList.items.length,
            isOptional: true,
            routineVersion: subroutine,
            translations: [{
                __typename: "NodeRoutineListItemTranslation" as const,
                id: uuid(),
                language,
                description: getTranslation(subroutine, [language])?.description,
                name: getTranslation(subroutine, [language])?.name,
            }],
        };
    }
}

/**
 * Manages adding, updating, and deleting nodes in a routine version.
 */
class NodeManager {
    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    public static DOM_ID_PREFIX = "node-";

    /**
     * Adds a node to the routineVersion.
     * 
     * @param routine The current routine version
     * @param node The node to create
     * @returns A new routineVersion with the created node.
     */
    static create(routine: RoutineVersionShape, node: Node): RoutineVersionShape {
        // const nodes = routine.nodes ?? [];

        // return {
        //     ...routine,
        //     nodes: addToArray(nodes, node),
        // };
        return {} as any;
    }

    /**
     * Updates a single node within a routineVersion.
     * 
     * @param routine The current routine version
     * @param node The node to update (must have an id that matches an existing node)
     * @returns A new routineVersion with the updated node.
     */
    static update(routine: RoutineVersionShape, node: Node): RoutineVersionShape {
        // const nodes = routine.nodes ?? [];
        // const nodeIndex = nodes.findIndex(n => n.id === node.id);
        // if (nodeIndex === -1) return routine;

        // return {
        //     ...routine,
        //     nodes: updateArray(nodes, nodeIndex, node),
        // };
        return {} as any;
    }

    /**
     * Deletes a node from the routineVersion.
     * 
     * @param routine The current routine version
     * @param node The node to delete
     * @returns A new routineVersion without the specified node.
     */
    static delete(routine: RoutineVersionShape, node: Pick<Node, "id">): RoutineVersionShape {
        // const nodes = routine.nodes ?? [];
        // const nodeIndex = nodes.findIndex(n => n.id === node.id);
        // if (nodeIndex === -1) return routine;

        // // Handle links that reference the node
        // const newLinks: ReturnType<typeof Graph.generate.link>[] = [];
        // const deletingLinks = routine.nodeLinks?.filter(l => l.from.id === node.id || l.to.id === node.id) ?? [];
        // // Find all "from" and "to" nodes in the deleting links
        // const fromNodeIds = deletingLinks.map(l => l.from.id).filter(id => id !== node.id);
        // const toNodeIds = deletingLinks.map(l => l.to.id).filter(id => id !== node.id);
        // // If there is only one "from" node, create a link between it and every "to" node
        // if (fromNodeIds.length === 1) {
        //     toNodeIds.forEach(toId => { newLinks.push(Graph.generate.link(fromNodeIds[0], toId)); });
        // }
        // // If there is only one "to" node, create a link between it and every "from" node
        // else if (toNodeIds.length === 1) {
        //     fromNodeIds.forEach(fromId => { newLinks.push(Graph.generate.link(fromId, toNodeIds[0])); });
        // }
        // // NOTE: Every other case is ambiguous, so we can't auto-create create links
        // // Delete old links
        // const keptLinks = routine.nodeLinks?.filter(l => !deletingLinks.includes(l)) ?? [];

        // return {
        //     ...routine,
        //     nodes: deleteArrayIndex(nodes, nodeIndex),
        //     nodeLinks: keptLinks.concat(newLinks),
        // };
        return {} as any;
    }

    static operate(routine: RoutineVersionShape, node: Node, operation: GraphElementOperation): RoutineVersionShape {
        switch (operation) {
            case "create":
                return this.create(routine, node);
            case "update":
                return this.update(routine, node);
            case "delete":
                return this.delete(routine, node);
            default:
                return routine;
        }
    }

    static isNodeEnd(node: Node): node is NodeEnd {
        return node.nodeType === NodeType.End;
    }

    static isNodeRoutineList(node: Node): node is NodeRoutineList {
        return node.nodeType === NodeType.RoutineList;
    }

    static isNodeStart(node: Node): node is NodeStart {
        // return node.nodeType === NodeType.Start;
        return false;
    }

    static find(routine: RoutineVersionShape, nodeId: string): Node | undefined {
        // const node = routine.nodes?.find(n => n.id === nodeId);
        // return node;
        return undefined;
    }

    /**
     * @param node The node to get the DOM ID for
     * @param suffix An optional suffix to append to the ID. Useful 
     * for sub-elements of the node
     * @returns The DOM ID for the node, if it exists
     */
    static getElementId(node: Pick<Node, "id">, suffix?: string): string {
        return `${this.DOM_ID_PREFIX}${node.id}${suffix ? `-${suffix}` : ""}`;
    }
}

/**
 * Manages adding, updating, and deleting links between nodes in a routine version.
 */
class LinkManager {
    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    /**
     * Creates a link between two nodes in the routineVersion.
     * 
     * @param routine The current routine version
     * @param link The link to create
     * @returns A new routineVersion with the created link.
     */
    static create(routine: RoutineVersionShape, link: NodeLink): RoutineVersionShape {
        // const nodeLinks = routine.nodeLinks ?? [];

        // return {
        //     ...routine,
        //     nodeLinks: addToArray(nodeLinks, link),
        // };
        return {} as any;
    }

    /**
     * Updates a link in the routineVersion.
     * 
     * @param routine The current routine version
     * @param link The link to update
     * @returns A new routineVersion with the updated link.
     */
    static update(routine: RoutineVersionShape, link: NodeLink): RoutineVersionShape {
        // const nodeLinks = routine.nodeLinks ?? [];
        // const nodeLinkIndex = nodeLinks.findIndex(n => n.id === link.id);
        // if (nodeLinkIndex !== -1) return routine;

        // return {
        //     ...routine,
        //     nodeLinks: updateArray(nodeLinks, nodeLinkIndex, link),
        // };
        return {} as any;
    }

    /**
     * Deletes a link from the routineVersion without deleting any nodes.
     * 
     * @param routine The current routine version
     * @param link The link to delete
     * @returns A new routineVersion without the specified link.
     */
    static delete(routine: RoutineVersionShape, link: Pick<NodeLink, "id">): RoutineVersionShape {
        // const nodeLinks = routine.nodeLinks ?? [];
        // const nodeLinkIndex = nodeLinks.findIndex(n => n.id === link.id);
        // if (nodeLinkIndex === -1) return routine;

        // return {
        //     ...routine,
        //     nodeLinks: deleteArrayIndex(nodeLinks, nodeLinkIndex),
        // };
        return {} as any;
    }

    static operate(routine: RoutineVersionShape, link: NodeLink, operation: GraphElementOperation): RoutineVersionShape {
        switch (operation) {
            case "create":
                return this.create(routine, link);
            case "update":
                return this.update(routine, link);
            case "delete":
                return this.delete(routine, link);
            default:
                return routine;
        }
    }

    static find(routine: RoutineVersionShape, linkId: string): NodeLink | undefined {
        // const link = routine.nodeLinks?.find(l => l.id === linkId);
        // return link;
        return undefined;
    }

    static findMany(routine: RoutineVersionShape, predicate: ((link: NodeLink) => boolean)) {
        // return routine.nodeLinks?.filter(predicate) ?? [];
        return [];
    }
}

/**
 * Manages graph operations for a routine version, such as reordering nodes and adding branches
 */
class OperationManager {
    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    /**
     * Inserts a node along a link. This removes the original link,
     * adds the new node, then links from the old link's 'from' node
     * to the new node, and from the new node to the old link's 'to' node.
     *
     * @param routine The current routine version.
     * @param link The link along which the node should be inserted.
     * @param newNode The new node to insert.
     */
    static insertNodeAlongLink(
        routine: RoutineVersionShape,
        link: NodeLink,
        newNode: Node,
    ): RoutineVersionShape {
        // Remove the old link
        routine = Graph.link.delete(routine, link);

        // Add the new node to the routine
        routine = Graph.node.create(routine, newNode);

        const fromId = link.from.id;
        const toId = link.to.id;

        // Create a new link from the original 'from' to the new node
        const newLink1 = Graph.generate.link(fromId, newNode.id);
        routine = Graph.link.create(routine, newLink1);

        // Create a new link from the new node to the original 'to'
        const newLink2 = Graph.generate.link(newNode.id, toId);
        routine = Graph.link.create(routine, newLink2);

        return routine;
    }

    /**
     * Moves a node so that it appears before the node it is dropped on.
     *
     * Steps:
     * 1. Delete `nodeToMove` from current position (fixes old references).
     * 2. Re-add `nodeToMove`.
     * 3. For each incoming link to `dropOnNode`, redirect it to `nodeToMove`.
     * 4. Add a link from `nodeToMove` to `dropOnNode`.
     *
     * Result:
     * Any path that led into `dropOnNode` now leads into `nodeToMove`, and then goes to `dropOnNode`.
     * 
     * @param routine The current routine version.
     * @param nodeToMove The node that was dragged and dropped.
     * @param dropOnNode The node that `nodeToMove` was dropped on.
     */
    static moveNode(
        routine: RoutineVersionShape,
        nodeToMove: Node,
        dropOnNode: Node,
    ): RoutineVersionShape {
        // // Remove the node from its old position
        // routine = Graph.node.delete(routine, nodeToMove);

        // // Re-add the node to the routine
        // routine = Graph.node.create(routine, nodeToMove);

        // // Find all links that currently point TO the dropOnNode
        // const incomingLinks = (routine.nodeLinks ?? []).filter(
        //     (l) => l.to.id === dropOnNode.id,
        // );

        // // Redirect each incoming link to nodeToMove instead of dropOnNode
        // for (const incomingLink of incomingLinks) {
        //     // Delete the old link pointing to dropOnNode
        //     routine = Graph.link.delete(routine, incomingLink);

        //     // Create a new link that points to nodeToMove
        //     const redirectedLink = Graph.generate.link(incomingLink.from.id, nodeToMove.id);
        //     routine = Graph.link.create(routine, redirectedLink);
        // }

        // // Now that nodeToMove has all the incoming links that dropOnNode had,
        // // we create a single link from nodeToMove to dropOnNode.
        // const bridgeLink = Graph.generate.link(nodeToMove.id, dropOnNode.id);
        // routine = Graph.link.create(routine, bridgeLink);

        // return routine;
        return {} as any;
    }

    /**
     * Branches the graph from a given link by creating a new node (of a specified type)
     * and attaching it to the same "from" as the existing link. This does not remove
     * the existing link, effectively creating a branch.
     *
     * For example:
     * 
     * fromNode -> originalTarget
     *
     * After branching with a new node:
     *
     *              -> originalTarget
     * fromNode ->
     *              -> newNode
     *
     * @param routine The current routine version.
     * @param link The link from which we want to branch.
     * @param nodeType The type of node to create for the new branch.
     * @param language The language to use for the new node's translation.
     */
    static branch(
        routine: RoutineVersionShape,
        link: NodeLink,
        nodeType: NodeType,
        language: string,
    ): RoutineVersionShape {
        let newNode: Node;
        // Create a new node of the specified type
        switch (nodeType) {
            case NodeType.End:
                newNode = Graph.generate.nodeEnd(language);
                break;
            case NodeType.RoutineList:
                newNode = Graph.generate.nodeRoutineList(routine, language);
                break;
            // Handle other node types if applicable
            default:
                // Fallback to End node if unrecognized. Adjust as needed.
                newNode = Graph.generate.nodeEnd(language);
                break;
        }

        // Add the new node to the routine
        routine = Graph.node.create(routine, newNode);

        // Create a new link from the link.from node to the new node
        const branchLink = Graph.generate.link(link.from.id, newNode.id);
        routine = Graph.link.create(routine, branchLink);

        return routine;
    }

    /**
    * Merges two nodes into one, effectively combining multiple paths.
    *
    * Triggered from a link, the user selects another node to merge with 
    * and decides which node to keep:
    * - If keepMergeNode is true, we keep `mergeNode` and remove `link.to`.
    * - If keepMergeNode is false, we keep `link.to` and remove `mergeNode`.
    *
    * All incoming links to the node-to-remove are re-routed to the node-to-keep,
    * and all outgoing links from the node-to-remove are also re-routed 
    * from the node-to-keep. Finally, the node-to-remove is deleted.
    */
    static merge(
        routine: RoutineVersionShape,
        link: NodeLink,
        mergeNode: Node,
        keepMergeNode: boolean,
    ): RoutineVersionShape {
        return {} as any;
        // const nodeToRemove = keepMergeNode ? link.to : mergeNode;
        // const nodeToKeep = keepMergeNode ? mergeNode : link.to;

        // // Redirect incoming links of nodeToRemove to nodeToKeep
        // const incomingLinks = (routine.nodeLinks ?? []).filter(
        //     (l) => l.to.id === nodeToRemove.id,
        // );

        // for (const incomingLink of incomingLinks) {
        //     // Remove old link
        //     routine = Graph.link.delete(routine, incomingLink);
        //     // Create new link to nodeToKeep if not linking a node to itself
        //     if (incomingLink.from.id !== nodeToKeep.id) {
        //         const newLink = Graph.generate.link(incomingLink.from.id, nodeToKeep.id);
        //         routine = Graph.link.create(routine, newLink);
        //     }
        // }

        // // Redirect outgoing links of nodeToRemove to originate from nodeToKeep
        // const outgoingLinks = (routine.nodeLinks ?? []).filter(
        //     (l) => l.from.id === nodeToRemove.id,
        // );

        // for (const outgoingLink of outgoingLinks) {
        //     // Remove old link
        //     routine = Graph.link.delete(routine, outgoingLink);
        //     // Create new link from nodeToKeep if not linking a node to itself
        //     if (outgoingLink.to.id !== nodeToKeep.id) {
        //         const newLink = Graph.generate.link(nodeToKeep.id, outgoingLink.to.id);
        //         routine = Graph.link.create(routine, newLink);
        //     }
        // }

        // // Remove the node we decided not to keep
        // routine = Graph.node.delete(routine, nodeToRemove);

        // return routine;
    }
}

/**
 * Manages adding, updating, and deleting routine list items within routine list nodes.
 */
class RoutineListItemManager {
    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    /**
     * Sorts routine list items by their index, and normalizes the index values
     * 
     * @param items The routine list items to sort
     * @returns The sorted routine list items
     */
    static sort(items: RoutineListItem[]): RoutineListItem[] {
        return items.sort((a, b) => a.index - b.index).map((item, index) => ({ ...item, index }));
    }

    /**
     * Adds a routine list item object to the routine list node.
     */
    static create(
        routine: RoutineVersionShape,
        nodeId: string,
        item: RoutineListItem,
    ): RoutineVersionShape {
        const node = Graph.node.find(routine, nodeId);
        if (!node || !Graph.node.isNodeRoutineList(node)) return routine;
        const items = node.routineList?.items ?? [];

        const updatedNode = {
            ...node,
            routineList: {
                ...node.routineList,
                items: this.sort(addToArray(items, item)),
            },
        };
        return Graph.node.update(routine, updatedNode);
    }

    /**
     * Updates a routine list item object within the routine list node.
     */
    static update(
        routine: RoutineVersionShape,
        nodeId: string,
        item: RoutineListItem,
    ): RoutineVersionShape {
        const node = Graph.node.find(routine, nodeId);
        if (!node || !Graph.node.isNodeRoutineList(node)) return routine;
        const items = node.routineList?.items ?? [];
        const itemIndex = items.findIndex(i => i.id === item.id);
        if (itemIndex === -1) return routine;

        const updatedNode = {
            ...node,
            routineList: {
                ...node.routineList,
                items: this.sort(updateArray(items, itemIndex, item)),
            },
        };
        return Graph.node.update(routine, updatedNode);
    }

    /**
     * Deletes a routine list item object from the routine list node.
     */
    static delete(
        routine: RoutineVersionShape,
        nodeId: string,
        item: Pick<RoutineListItem, "id">,
    ): RoutineVersionShape {
        const node = Graph.node.find(routine, nodeId);
        if (!node || !Graph.node.isNodeRoutineList(node)) return routine;
        const items = node.routineList?.items ?? [];
        const itemIndex = items.findIndex(i => i.id === item.id);
        if (itemIndex === -1) return routine;

        const updatedNode = {
            ...node,
            routineList: {
                ...node.routineList,
                items: this.sort(deleteArrayIndex(items, itemIndex)),
            },
        };
        return Graph.node.update(routine, updatedNode);
    }

    static operate(
        routine: RoutineVersionShape,
        nodeId: string,
        item: RoutineListItem,
        operation: GraphElementOperation,
    ): RoutineVersionShape {
        switch (operation) {
            case "create":
                return this.create(routine, nodeId, item);
            case "update":
                return this.update(routine, nodeId, item);
            case "delete":
                return this.delete(routine, nodeId, item);
            default:
                return routine;
        }
    }
}

type RenderProps = any;
//     handleLinkOperation: (operation: LinkOperation, link: NodeLink) => unknown;
//     handleNodeOperation: (operation: GraphElementOperation, node: Node) => unknown;
//     handleNodeSelect: (node: SomeNode) => unknown;
//     handleRoutineListItemOperation: (operation: GraphElementOperation, nodeId: string, item: RoutineListItem) => unknown;
//     /** True if the user is able to edit the graph */
//     isEditing: boolean,
//     /** The user's current language */
//     language: string,
//     /**
//      * The location to start rendering from. Since we're dealing 
//      * with a single graph, we can safely use one number instead of 
//      * an array.
//      */
//     location: number,
//     /** All graph status messages */
//     messages: GraphMessageWithStatus[],
//     /** The root step of the graph */
//     rootStep: MultiRoutineStep,
//     /** The overall routine version being rendered */
//     routine: RoutineVersionShape,
//     /** The scale of the graph */
//     scale: number,
//     /** The currently selected node */
//     selectedNode: SomeNode | null;
//     /** A set of visited locations to prevent infinite loops. Changes as we recurse */
//     visited?: Set<string>,
// }

export class Graph {
    public static generate = GenerateManager;
    public static link = LinkManager;
    public static node = NodeManager;
    public static operation = OperationManager;
    public static routineListItem = RoutineListItemManager;

    /**
    * Multi-step routine initial data, if creating from scratch
    * @param session The current session
    * @param existing The existing routine version, if editing
    * @returns Initial data for a routine being created or updated
    */
    static initialize(
        session: Session | undefined,
        existing?: Partial<RoutineVersion> | null | undefined,
    ): RoutineVersionShape {
        // let result = routineSingleStepInitialValues(session, existing);
        // result.routineType = RoutineType.MultiStep;
        // const shouldGenerateGraph = !existing || !existing.nodes || !existing.nodeLinks;
        // if (!shouldGenerateGraph) {
        //     return {
        //         ...result,
        //         nodes: existing.nodes,
        //         nodeLinks: existing.nodeLinks,
        //     };
        // }

        // result = this.addSwimLane(session, result);
        // return result;
        return {} as any;
    }

    /**
     * Initializes a new swim lane graph for a routine version.
     * @param session The current session
     * @param routineVersion The routine version to add the swim lane to
     * @returns Updated routine version with a new swim lane graph
     */
    static addSwimLane(
        session: Session | undefined,
        routineVersion: RoutineVersionShape,
    ): RoutineVersionShape {
        // const language = getUserLanguages(session)[0];
        // const startNode: NodeStart = {
        //     __typename: "Node" as const,
        //     id: uuid(),
        //     nodeType: NodeType.Start,
        //     columnIndex: 0, // Deprecated
        //     rowIndex: 0, // Deprecated
        //     translations: [],
        // };
        // const routineListNodeId = uuid();
        // const routineListNode: NodeRoutineList = {
        //     __typename: "Node",
        //     id: routineListNodeId,
        //     nodeType: NodeType.RoutineList,
        //     columnIndex: 0, // Deprecated
        //     rowIndex: 0, // Deprecated
        //     routineList: {
        //         __typename: "NodeRoutineList",
        //         id: uuid(),
        //         isOptional: false,
        //         isOrdered: false,
        //         items: [],
        //     },
        //     translations: [{
        //         __typename: "NodeTranslation",
        //         id: uuid(),
        //         language,
        //         name: "Subroutine",
        //     }] as Node["translations"],
        // };
        // const endNodeId = uuid();
        // const endNode: NodeEnd = {
        //     __typename: "Node",
        //     id: endNodeId,
        //     nodeType: NodeType.End,
        //     columnIndex: 0, // Deprecated
        //     rowIndex: 0, // Deprecated
        //     end: {
        //         __typename: "NodeEnd",
        //         id: uuid(),
        //         wasSuccessful: true,
        //     },
        //     translations: [{
        //         __typename: "NodeTranslation" as const,
        //         id: uuid(),
        //         language,
        //         name: "End",
        //         description: "",
        //     }],
        // };
        // const link1: NodeLink = {
        //     __typename: "NodeLink",
        //     id: uuid(),
        //     from: startNode,
        //     to: routineListNode,
        //     whens: [],
        //     operation: null,
        // };
        // const link2: NodeLink = {
        //     __typename: "NodeLink",
        //     id: uuid(),
        //     from: routineListNode,
        //     to: endNode,
        //     whens: [],
        //     operation: null,
        // };

        // const result = {
        //     ...routineVersion,
        //     nodes: [...(routineVersion.nodes ?? []), startNode, routineListNode, endNode],
        //     nodeLinks: [...(routineVersion.nodeLinks ?? []), link1, link2],
        // };
        // return result;
        return {} as any;
    }

    /**
    * Recursively renders the graph starting at a given location.
    * We will:
    * - Render the current step.
    * - If it's a DecisionStep, render each option and recurse down each branch.
    * - If it's an EndStep, render the end node and stop recursing.
    * - Otherwise, find the nextLocation and recurse until there's no next step.
    * 
    * @returns The rendered graph
    */
    static render({
        visited = new Set<string>(),
        ...props
    }: RenderProps): React.ReactNode {
        // const {
        //     handleLinkOperation,
        //     handleNodeOperation,
        //     handleNodeSelect,
        //     handleRoutineListItemOperation,
        //     isEditing,
        //     language,
        //     location,
        //     messages,
        //     rootStep,
        //     routine,
        //     scale,
        //     selectedNode,
        // } = props;
        // // Use the location to find the current step
        // const currentStep = rootStep.nodes[location];
        // console.log("qqqq starting render", location, currentStep);
        // if (!currentStep) return null;

        // // Create a unique key for visited checks. We'll use the location array joined by '-'.
        // const locationKey = location + "";
        // if (visited.has(locationKey)) {
        //     // Already visited this step at this location, avoid infinite loops
        //     return null;
        // }
        // visited.add(locationKey);

        // // Render the current node
        // let renderedStep: React.ReactNode;
        // const nodeId: string | undefined = (currentStep as { nodeId?: string }).nodeId;
        // const commonProps = {
        //     isEditing,
        //     isLinked: true, // Must be true since we used the graph to find this node
        //     isSelected: Boolean(nodeId) && selectedNode?.id === nodeId,
        //     language,
        //     messages: messages.filter(m => m.nodeId === nodeId),
        //     onSelect: handleNodeSelect,
        //     scale,
        // } as const;
        // if (commonProps.isSelected) {
        //     console.log("in render. Node isssss selected...", selectedNode);
        // }
        // switch (currentStep.__type) {
        //     case RunStepType.End: {
        //         const currentNode = Graph.node.find(routine, currentStep.nodeId);
        //         if (!currentNode || !Graph.node.isNodeEnd(currentNode)) return null;

        //         renderedStep = (
        //             <EndNode
        //                 handleAction={handleNodeOperation}
        //                 node={currentNode}
        //                 {...commonProps}
        //             />
        //         );
        //         break;
        //     }
        //     case RunStepType.RoutineList: {
        //         const currentNode = Graph.node.find(routine, currentStep.nodeId);
        //         if (!currentNode || !Graph.node.isNodeRoutineList(currentNode)) return null;

        //         renderedStep = (
        //             <RoutineListNode
        //                 handleAction={handleNodeOperation}
        //                 handleSubroutineAction={handleRoutineListItemOperation}
        //                 node={currentNode}
        //                 {...commonProps}
        //             />
        //         );
        //         break;
        //     }
        //     // For now, everything else gets skipped
        //     default: {
        //         renderedStep = null;
        //         break;
        //     }
        // }

        // // If the current step is a DecisionStep, there is a branch in the graph. 
        // // Recurse down each branch to render the entire graph.
        // if (currentStep.__type === RunStepType.Decision) {
        //     const links = currentStep.options.map(o => o.link);

        //     // Render GraphLink for branching
        //     // GraphLink handles multiple links and places them accordingly
        //     const junctionLink = (
        //         <GraphLink
        //             key={`links-${locationKey}`}
        //             links={links}
        //             isEditing={isEditing}
        //             onAction={handleLinkOperation}
        //         />
        //     );

        //     // Render each branch
        //     const branches = currentStep.options.map((option, idx) => {
        //         const optionLocation = option.step.location;
        //         const nextLocation = optionLocation[optionLocation.length - 1];
        //         const optionSubtree = this.render({
        //             ...props,
        //             location: nextLocation,
        //             visited,
        //         });
        //         return (
        //             <div key={`branch-${idx}`}>
        //                 {optionSubtree}
        //             </div>
        //         );
        //     });

        //     return (
        //         <>
        //             <Box mb={16}>
        //                 {renderedStep}
        //             </Box>
        //             {junctionLink}
        //             {branches}
        //         </>
        //     );
        // }
        // // Otherwise, find the next step and recurse
        // else {
        //     // If not an EndStep, attempt to find the next step
        //     // const nextLocationArray = currentStep.__type !== RunStepType.End
        //     //     ? new RunStepNavigator(console).getNextLocation([...ROOT_LOCATION, location], rootStep)
        //     //     : undefined;
        //     const nextLocationArray = []; //TODO
        //     const nextLocation = nextLocationArray ? nextLocationArray[nextLocationArray.length - 1] : -1;
        //     if (nextLocation >= 0) {
        //         const links = Graph.link.findMany(routine, (link) => link.from.id === (currentStep as { nodeId: string }).nodeId);
        //         // Render edge from current to next
        //         const edge = renderedStep ? (
        //             <GraphLink
        //                 key={`links-${locationKey}`}
        //                 links={links}
        //                 isEditing={isEditing}
        //                 onAction={handleLinkOperation}
        //             />
        //         ) : null;
        //         const nextSubtree = this.render({
        //             ...props,
        //             location: nextLocation,
        //             visited,
        //         });
        //         return (
        //             <>
        //                 {renderedStep ? <Box mb={8}>
        //                     {renderedStep}
        //                 </Box> : null}
        //                 {edge}
        //                 {nextSubtree}
        //             </>
        //         );
        //     } else {
        //         // No next step, just return the node
        //         return renderedStep;
        //     }
        // }
        return null;
    }
}









// const initialBpmnXml = `<?xml version="1.0" encoding="UTF-8"?>
// <definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"
//              targetNamespace="http://bpmn.io/schema/bpmn"
//              xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
//              xmlns:omgdc="http://www.omg.org/spec/DD/20100524/DC"
//              xmlns:omgdi="http://www.omg.org/spec/DD/20100524/DI">
//   <process id="Process_1" isExecutable="false" />
//   <bpmndi:BPMNDiagram id="BPMNDiagram_1">
//     <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_1" />
//   </bpmndi:BPMNDiagram>
// </definitions>`;
const exampleBpmnXml = `<?xml version="1.0" encoding="UTF-8"?>
<definitions
  xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
  xmlns:omgdc="http://www.omg.org/spec/DD/20100524/DC"
  xmlns:omgdi="http://www.omg.org/spec/DD/20100524/DI"
  xsi:schemaLocation="http://www.omg.org/spec/BPMN/20100524/MODEL 
                      http://www.omg.org/spec/BPMN/20100524/BPMN20.xsd"
  typeLanguage="http://www.w3.org/2001/XMLSchema"
  targetNamespace="http://bpmn.io/schema/bpmn"
  expressionLanguage="http://www.w3.org/1999/XPath"
  id="EmptyDefinition">
  
  <!-- The process with a start event -->
  <process id="Process_1" isExecutable="false">
    <startEvent id="StartEvent_1" name="Start" />
  </process>

  <!-- The diagram containing a plane for Process_1,
       plus a shape for the start event -->
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_1">
      <bpmndi:BPMNShape id="StartEvent_1_di" bpmnElement="StartEvent_1">
        <omgdc:Bounds x="186" y="136" width="36" height="36" />
      </bpmndi:BPMNShape>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</definitions>
`;

interface BpmnModelerProps {
    onChange?: (xml: string) => unknown;
    xml: string;
}

export function BpmnModeler({
    onChange,
    xml,
}: BpmnModelerProps) {
    // const containerRef = useRef<HTMLDivElement | null>(null);
    // const modelerRef = useRef<Modeler | null>(null);

    // useEffect(function setupModelerEffect() {
    //     if (!containerRef.current) return;

    //     modelerRef.current = new Modeler({
    //         container: containerRef.current,
    //     });

    //     return function cleanupModeler() {
    //         modelerRef.current?.destroy();
    //     };
    // }, []);

    // useEffect(function importXmlEffect() {
    //     if (!modelerRef.current) return;

    //     modelerRef.current.importXML(xml).then(() => {
    //         if (!modelerRef.current) return;

    //         // If you want to handle changes from the user editing the diagram,
    //         // you can hook into the modeler's 'commandStack.changed' event, for example.
    //         modelerRef.current.on("commandStack.changed", async function handleCommandStackChanged() {
    //             if (!modelerRef.current) return;
    //             const { xml } = await modelerRef.current.saveXML();
    //             if (xml && typeof onChange === "function") {
    //                 onChange(xml);
    //             }
    //         });

    //         const canvas = modelerRef.current.get("canvas") as Canvas;
    //         if (!canvas) return;
    //         canvas.zoom("fit-viewport");
    //     }).catch(err => console.error(err));
    // }, [xml, onChange]);

    // return <div ref={containerRef} style={{ width: "100%", height: "70vh" }} />;
    return null;
}












const MultiStepButtonsOuter = styled(Box)(({ theme }) => ({
    alignItems: "center",
    background: "transparent",
    display: "flex",
    justifyContent: "center",
    position: "absolute",
    zIndex: theme.spacing(2),
    bottom: 0,
    right: 0,
    paddingBottom: "calc(16px + env(safe-area-inset-bottom))",
    paddingLeft: "calc(16px + env(safe-area-inset-left))",
    paddingRight: "calc(16px + env(safe-area-inset-right))",
    height: "calc(64px + env(safe-area-inset-bottom))",
    width: "min(100%, 800px)",
}));

interface ZoomButtonsProps {
    handleScaleChange: (updatedScale: number) => unknown;
    scale: number;
}

const ZoomButtonsOuter = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    background: theme.palette.background.paper,
    borderRadius: theme.spacing(1),
    width: "fit-content",
    margin: theme.spacing(1),
    boxShadow: theme.shadows[4],
    position: "absolute",
    left: 0,
    bottom: pagePaddingBottom,
}));

const dividerStyle = { width: "100%" } as const;

function ZoomButtons({
    handleScaleChange,
    scale,
}: ZoomButtonsProps) {
    const { palette } = useTheme();

    const iconButtonStyle = { color: palette.background.textPrimary } as const;

    const zoomIn = useCallback(function zoomInCallback() {
        handleScaleChange(scale * (1 + SCALE_FACTOR));
    }, [handleScaleChange, scale]);

    const zoomOut = useCallback(function zoomOutCallback() {
        handleScaleChange(scale * (1 - SCALE_FACTOR));
    }, [handleScaleChange, scale]);

    return (
        <ZoomButtonsOuter>
            <IconButton
                aria-label={"Zoom In"}
                onClick={zoomIn}
                size="small"
                sx={iconButtonStyle}
            >
                <IconCommon name="Add" />
            </IconButton>
            <Divider sx={dividerStyle} />
            <IconButton
                aria-label={"Zoom Out"}
                onClick={zoomOut}
                size="small"
                sx={iconButtonStyle}
            >
                <IconCommon name="Minus" />
            </IconButton>
        </ZoomButtonsOuter>
    );
}

interface FormButtonsProps {
    disabledCancel: boolean;
    disabledSubmit: boolean;
    errors: FormErrors;
    handleCancel: () => unknown;
    handleSubmit: () => unknown;
    isCreate: boolean;
    isEditing: boolean;
    loading: boolean;
}

function FormButtons({
    disabledCancel,
    disabledSubmit,
    errors,
    handleCancel,
    handleSubmit,
    isCreate,
    isEditing,
    loading,
}: FormButtonsProps) {
    const { t } = useTranslation();

    const { openPopover, Popover } = useErrorPopover({ errors, onSetSubmitting: undefined });

    const hasErrors = useMemo(() => Object.values(errors ?? {}).some((value) => exists(value)), [errors]);
    const isSubmitDisabled = useMemo(() => loading || hasErrors || !isEditing || disabledSubmit, [disabledSubmit, hasErrors, isEditing, loading]);

    const onSubmit = useCallback(function SonSubmitCallback(ev: React.MouseEvent | React.TouchEvent) {
        // If formik invalid, display errors in popup
        if (hasErrors) openPopover(ev);
        else if (isEditing && !disabledSubmit) handleSubmit();
    }, [disabledSubmit, handleSubmit, hasErrors, isEditing, openPopover]);

    const onCancel = useCallback(function onCancelCallback(event: React.MouseEvent | React.TouchEvent) {
        event.preventDefault();
        event.stopPropagation();
        handleCancel();
    }, [handleCancel]);

    if (!isEditing) return null;
    return (
        <MultiStepButtonsOuter>
            <Grid container spacing={1} ml={2}>
                <BottomActionsGrid display="Page">
                    <Popover />
                    <Grid item xs={6}>
                        <Box onClick={onSubmit}>
                            <LoadableButton
                                aria-label={t(isCreate ? "Create" : "Save")}
                                disabled={isSubmitDisabled}
                                isLoading={loading}
                                startIcon={<IconCommon name={isCreate ? "Create" : "Save"} />}
                                variant="contained"
                            >{t(isCreate ? "Create" : "Save")}</LoadableButton>
                        </Box>
                    </Grid>
                    <Grid item xs={6}>
                        <Button
                            aria-label={t("Cancel")}
                            disabled={loading || (disabledCancel !== undefined ? disabledCancel : false)}
                            fullWidth
                            onClick={onCancel}
                            startIcon={<IconCommon name="Cancel" />}
                            variant="outlined"
                        >{t("Cancel")}</Button>
                    </Grid>
                </BottomActionsGrid>
            </Grid>
        </MultiStepButtonsOuter>
    );
}

const SwimLaneOuter = styled(Box)(({ theme }) => ({
    display: "inline-block",
    verticalAlign: "top",
    border: `2px solid ${theme.palette.primary.main}`,
    background: theme.palette.background.default,
    marginBottom: "16px",
    overflow: "overlay",
    flexShrink: 0,
    flex: "0 0 auto",
    minHeight: "100vh",
}));

const SwimLaneTopBar = styled(Box)(({ theme }) => ({
    height: "48px",
    background: theme.palette.primary.main,
    color: theme.palette.primary.contrastText,
    display: "flex",
    flexDirection: "row",
    alignItems: "center",
    paddingLeft: theme.spacing(1),
    paddingRight: theme.spacing(1),
}));

const SwimLaneBody = styled(Box)(({ theme }) => ({
    padding: theme.spacing(2),
}));

const swimLaneTitleDisplayStyle = {
    flexGrow: 1,
    width: "auto",
    "& .MuiInputBase-input": {
        padding: 0,
    },
} as const;

interface SwimLaneProps {
    children: React.ReactNode;
    handleNodeOperation: (operation: GraphElementOperation, node: Node) => unknown;
    index: number;
    isEditing: boolean;
    startNode: NodeStart;
}

function SwimLane({
    children,
    handleNodeOperation,
    index,
    isEditing,
    startNode,
}: SwimLaneProps) {
    // const { palette, typography } = useTheme();
    // const session = useContext(SessionContext);
    // const languages = useMemo(() => getUserLanguages(session), [session]);

    // const { draggableId, title } = useMemo(() => {
    //     const draggableId = startNode.id;
    //     const title = getTranslation(startNode, languages).name ?? `Lane ${index + 1}`;

    //     return { draggableId, title };
    // }, [index, languages, startNode]);

    // const updateLabel = useCallback((updatedLabel: string) => {
    //     const translations = updateTranslationFields<NodeTranslationShape, NodeStart>(startNode, languages[0], { name: updatedLabel });
    //     handleNodeOperation("update", { ...startNode, translations });
    // }, [handleNodeOperation, languages, startNode]);
    // const {
    //     editedLabel,
    //     handleLabelChange,
    //     handleLabelKeyDown,
    //     isEditingLabel,
    //     labelEditRef,
    //     startEditingLabel,
    //     submitLabelChange,
    // } = useEditableLabel({
    //     isEditable: isEditing,
    //     isMultiline: false,
    //     label: title,
    //     onUpdate: updateLabel,
    // });

    // const textFieldInputProps = useMemo(function textFieldInputPropsMemo() {
    //     return { style: (typography["h6"] as object || {}) };
    // }, [typography]);
    // const labelStyle = useMemo(function labelStyleMemo() {
    //     return {
    //         color: palette.primary.contrastText,
    //         cursor: isEditing ? "pointer" : "default",
    //         // Ensure there's a clickable area, even if the text is empty
    //         minWidth: "20px",
    //         minHeight: "24px",
    //         "&:empty::before": {
    //             content: "\"\"",
    //             display: "inline-block",
    //             width: "100%",
    //             height: "100%",
    //         },
    //     };
    // }, [isEditing, palette.primary.contrastText]);

    // return (
    //     <Draggable
    //         draggableId={draggableId}
    //         index={index}
    //     >
    //         {(provided) => (
    //             <SwimLaneOuter
    //                 ref={provided.innerRef}
    //                 {...provided.draggableProps}
    //             >
    //                 <SwimLaneTopBar>
    //                     <legend aria-label={`Lane ${index + 1} title`}>
    //                         {isEditingLabel ? (
    //                             <TextField
    //                                 ref={labelEditRef}
    //                                 inputMode="text"
    //                                 InputProps={textFieldInputProps}
    //                                 onBlur={submitLabelChange}
    //                                 onChange={handleLabelChange}
    //                                 onKeyDown={handleLabelKeyDown}
    //                                 placeholder={"Enter lane title..."}
    //                                 size="small"
    //                                 value={editedLabel}
    //                                 sx={swimLaneTitleDisplayStyle}
    //                             />
    //                         ) : (
    //                             <Typography
    //                                 onClick={startEditingLabel}
    //                                 sx={labelStyle}
    //                                 variant="h6"
    //                             >
    //                                 {editedLabel}
    //                             </Typography>
    //                         )}
    //                     </legend>
    //                     <Box
    //                         {...provided.dragHandleProps}
    //                         ml="auto"
    //                     >
    //                         <DragIcon fill={palette.primary.contrastText} />
    //                     </Box>
    //                 </SwimLaneTopBar>
    //                 <SwimLaneBody>
    //                     {children}
    //                 </SwimLaneBody>
    //             </SwimLaneOuter>
    //         )}
    //     </Draggable>
    // );
    return null;
}

const AddLaneButtonOuter = styled(Box)(({ theme }) => ({
    display: "flex",
    verticalAlign: "top",
    border: `2px dashed ${theme.palette.primary.main}`,
    background: theme.palette.background.default,
    overflow: "hidden",
    width: "200px",
    minHeight: "100vh",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    cursor: "pointer",
    "&:hover": {
        backgroundColor: theme.palette.action.hover,
    },
    flexShrink: 0,
    flex: "0 0 auto",
}));

interface AddLaneButtonProps {
    onAddLane: () => unknown;
}

function AddLaneButton({
    onAddLane,
}: AddLaneButtonProps) {
    // const { palette } = useTheme();

    // return (
    //     <AddLaneButtonOuter onClick={onAddLane}>
    //         <Typography variant="body1" sx={{ color: palette.primary.main, fontWeight: "bold" }}>
    //             + Add new lane
    //         </Typography>
    //     </AddLaneButtonOuter>
    // );
    return null;
}

function transformRoutineVersionValues(values: RoutineVersionShape, existing: RoutineVersionShape, isCreate: boolean) {
    return isCreate ? shapeRoutineVersion.create(values) : shapeRoutineVersion.update(existing, values);
}

const DroppableContainer = styled(Box)(() => ({
    display: "flex",
    flexDirection: "row",
    flexWrap: "nowrap",
    alignItems: "flex-start",
    flexShrink: 0,
}));

const baseFormStyle = {
    margin: "0 auto",
    padding: 0,
    display: "flex",
    flexDirection: "row",
    alignItems: "flex-start",
    overflowX: "auto",
    overflowY: "hidden",
} as const;
const formSectionStyle = { overflowX: "hidden" } as const;
const titleDialogFormStyle = {
    padding: "16px",
    width: "600px",
} as const;

type RoutineMultiStepCrudFormProps = FormProps<RoutineVersion, RoutineVersionShape> & {
    graphErrors: FormErrors;
    handleAddSwimLane: () => unknown;
    handleLinkOperation: (operation: LinkOperation, link: NodeLink) => unknown;
    handleNodeOperation: (operation: GraphElementOperation, node: Node) => unknown;
    handleRoutineListItemOperation: (operation: GraphElementOperation, nodeId: string, item: RoutineListItem) => unknown;
    hasRoutineChanged: boolean;
    isEditing: boolean;
    status: StatusMessageArray;
}

function RoutineMultiStepCrudForm({
    // disabled,
    // display,
    // existing,
    // graphErrors,
    // handleAddSwimLane,
    // handleNodeOperation,
    // handleLinkOperation,
    // handleRoutineListItemOperation,
    // handleUpdate,
    // hasRoutineChanged,
    // isCreate,
    // isEditing,
    // isOpen,
    // isReadLoading,
    // onCancel,
    // onClose,
    // onCompleted,
    // onDeleted,
    // status,
    // values,
    ...props
}: RoutineMultiStepCrudFormProps) {
    // const session = useContext(SessionContext);
    // const { t } = useTranslation();
    // const { breakpoints } = useTheme();
    // const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    // const [activeView, setActiveView] = useState<"graph" | "info" | "comments" | "issues" | "reports">("graph");

    // const [selectedNode, setSelectedNode] = useState<SomeNode | null>(null);
    // const handleNodeSelect = useCallback((node: SomeNode) => {
    //     // Toggle selection
    //     setSelectedNode(s => s?.id === node.id ? null : node);
    // }, []);
    // console.log("selectedNode issssssss", selectedNode?.id?.substring(0, 8), values.nodes?.map(n => n.id.substring(0, 8)));

    // const {
    //     handleAddLanguage,
    //     handleDeleteLanguage,
    //     language,
    //     languages,
    //     setLanguage,
    //     translationErrors,
    // } = useTranslatedFields({
    //     defaultLanguage: getUserLanguages(session)[0],
    //     validationSchema: routineVersionTranslationValidation.create({ env: process.env.NODE_ENV }),
    // });

    // const { handleCancel, handleCompleted, handleDeleted } = useUpsertActions<RoutineVersion>({
    //     display,
    //     isCreate,
    //     objectId: values.id,
    //     objectType: "RoutineVersion",
    //     ...props,
    // });
    // const {
    //     fetch,
    //     isCreateLoading,
    //     isUpdateLoading,
    // } = useUpsertFetch<RoutineVersion, RoutineVersionCreateInput, RoutineVersionUpdateInput>({
    //     isCreate,
    //     isMutate: true,
    //     endpointCreate: endpointsRoutineVersion.createOne,
    //     endpointUpdate: endpointsRoutineVersion.updateOne,
    // });
    // useSaveToCache({ isCreate, values, objectId: values.id, objectType: "RoutineVersion" });

    // // Handle delete
    // const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    // const handleDelete = useCallback(() => {
    //     function performDelete() {
    //         fetchLazyWrapper<DeleteOneInput, Success>({
    //             fetch: deleteMutation,
    //             inputs: { id: values.id, objectType: DeleteType.RoutineVersion },
    //             successCondition: (data) => data.success,
    //             successMessage: () => ({ messageKey: "ObjectDeleted", messageVariables: { name: getDisplay(values).title ?? t("Routine", { count: 1 }) } }),
    //             onSuccess: () => { handleDeleted(values as RoutineVersion); },
    //             errorMessage: () => ({ messageKey: "FailedToDelete" }),
    //         });
    //     }
    //     PubSub.get().publish("alertDialog", {
    //         messageKey: "DeleteConfirm",
    //         buttons: [{
    //             labelKey: "Delete",
    //             onClick: performDelete,
    //         }, {
    //             labelKey: "Cancel",
    //         }],
    //     });
    // }, [deleteMutation, values, t, handleDeleted]);

    // const onSubmit = useSubmitHelper<RoutineVersionCreateInput | RoutineVersionUpdateInput, RoutineVersion>({
    //     disabled,
    //     existing,
    //     fetch,
    //     inputs: transformRoutineVersionValues(values, existing, isCreate),
    //     isCreate,
    //     onSuccess: (data) => { handleCompleted(data); },
    //     onCompleted: () => { props.setSubmitting(false); },
    // });

    // const [scale, setScale] = useState<number>(DEFAULT_SCALE_PERCENT);
    // // usePinchZoom({
    // //     initialScale: scale,
    // //     minScale: MIN_SCALE_PERCENT,
    // //     maxScale: MAX_SCALE_PERCENT,
    // //     onScaleChange: handleScaleChange,
    // //     validTargetIds: [ELEMENT_IDS.RoutineMultiStepCrudGraph, "node-", "subroutine"],
    // // });

    // const titleTopBarStyle = useMemo(function topBarStyleMemo() {
    //     return {
    //         stack: {
    //             padding: 0,
    //             ...(display === "Page" && !isMobile ? {
    //                 margin: "auto",
    //                 maxWidth: "700px",
    //                 paddingTop: 1,
    //                 paddingBottom: 1,
    //             } : {}),
    //         },
    //     } as const;
    // }, [display, isMobile]);

    // const titleDialogContentForm = useCallback(function titleDialogContentFormCallback() {
    //     return (
    //         <ScrollBox>
    //             <BaseForm
    //                 display="Dialog"
    //                 maxWidth={600}
    //                 style={titleDialogFormStyle}
    //             >
    //                 <FormContainer>
    //                     <RelationshipList
    //                         isEditing={!disabled}
    //                         objectType={"Routine"}
    //                         sx={{ marginBottom: 4 }}
    //                     />
    //                     <FormSection sx={formSectionStyle}>
    //                         <LanguageInput
    //                             currentLanguage={language}
    //                             handleAdd={handleAddLanguage}
    //                             handleDelete={handleDeleteLanguage}
    //                             handleCurrent={setLanguage}
    //                             languages={languages}
    //                         />
    //                         <TranslatedTextInput
    //                             fullWidth
    //                             label={t("Name")}
    //                             language={language}
    //                             name="name"
    //                         />
    //                         <TranslatedRichInput
    //                             language={language}
    //                             maxChars={2048}
    //                             minRows={4}
    //                             name="description"
    //                             placeholder={t("DescriptionPlaceholder")}
    //                         />
    //                     </FormSection>
    //                 </FormContainer>
    //             </BaseForm>
    //         </ScrollBox>
    //     );
    // }, [disabled, handleAddLanguage, handleDeleteLanguage, language, languages, setLanguage, t]);

    // const { messages, rootStep } = useMemo(function buildRootStepsMemo() {
    //     // const rootStep = new RunStepBuilder(languages, console).runnableObjectToStep(values as unknown as RunnableRoutineVersion, [...ROOT_LOCATION]);
    //     // const { messages } = routineVersionStatusMultiStep(values as unknown as RunnableRoutineVersion, rootStep);
    //     // return { messages, rootStep };
    //     return {} as any;
    // }, [values, languages]);

    // function onDragEnd(result: DropResult) {
    //     //TODO
    //     console.log("qqqq onDragEnd", result);
    // }

    // const isLoading = useMemo(() => isCreateLoading || isDeleteLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isDeleteLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    // return (
    //     <MaybeLargeDialog
    //         display={display}
    //         id={ELEMENT_IDS.RoutineMultiStepCrudDialog}
    //         isOpen={isOpen}
    //         onClose={onClose}
    //     >
    //         <Box
    //             id={ELEMENT_IDS.RoutineMultiStepCrudGraph}
    //             display="flex"
    //             flexDirection="column"
    //             minHeight="100vh"
    //         >
    //             <TopBar
    //                 display={display}
    //                 onClose={onClose}
    //                 // TODO needs to modify changestack
    //                 titleComponent={<EditableTitle
    //                     handleDelete={handleDelete}
    //                     isDeletable={!(isCreate || disabled)}
    //                     isEditable={!disabled}
    //                     language={language}
    //                     titleField="name"
    //                     subtitleField="description"
    //                     variant="subheader"
    //                     sxs={titleTopBarStyle}
    //                     DialogContentForm={titleDialogContentForm}
    //                 />}
    //             />
    //             {/* <DragDropContext onDragEnd={onDragEnd}>
    //                 <BaseForm
    //                     display={display}
    //                     isLoading={isLoading}
    //                     maxWidth={"100%"}
    //                     style={baseFormStyle}
    //                 >
    //                     <Droppable droppableId="graph">
    //                         {(provided) => (
    //                             <DroppableContainer
    //                                 ref={provided.innerRef}
    //                                 {...provided.droppableProps}
    //                             >
    //                                 {
    //                                     rootStep?.__type === RunStepType.MultiRoutine ? (
    //                                         rootStep.startNodeIndexes.map((startIndex, i) => {
    //                                             // Get the start step for this swim lane
    //                                             const startStep = rootStep.nodes[startIndex];
    //                                             if (!startStep || startStep.__type !== RunStepType.Start) return null;

    //                                             const startNodeId = (startStep as StartStep).nodeId;
    //                                             const startNode = Graph.node.find(values, startNodeId);
    //                                             if (!startNode || !Graph.node.isNodeStart(startNode)) return null;

    //                                             return (
    //                                                 <SwimLane
    //                                                     handleNodeOperation={handleNodeOperation}
    //                                                     index={i}
    //                                                     isEditing={isEditing}
    //                                                     key={i}
    //                                                     startNode={startNode}
    //                                                 >
    //                                                     {Graph.render({
    //                                                         handleLinkOperation,
    //                                                         handleNodeOperation,
    //                                                         handleNodeSelect,
    //                                                         handleRoutineListItemOperation,
    //                                                         isEditing,
    //                                                         language,
    //                                                         location: startIndex,
    //                                                         messages,
    //                                                         rootStep,
    //                                                         routine: values,
    //                                                         scale,
    //                                                         selectedNode,
    //                                                     })}
    //                                                 </SwimLane>
    //                                             );
    //                                         })
    //                                     ) : (
    //                                         // If it's not a MultiRoutineStep, either render nothing or handle single-lane scenario
    //                                         null
    //                                     )
    //                                 }
    //                                 {provided.placeholder}
    //                                 <AddLaneButton onAddLane={handleAddSwimLane} />
    //                             </DroppableContainer>
    //                         )}
    //                     </Droppable>
    //                 </BaseForm>
    //             </DragDropContext> */}
    //             <ZoomButtons
    //                 handleScaleChange={setScale}
    //                 scale={scale}
    //             />
    //             {/* <BpmnModeler xml={exampleBpmnXml} onChange={noop} /> */}
    //             <FormButtons
    //                 disabledCancel={isLoading}
    //                 disabledSubmit={isLoading || !hasRoutineChanged}
    //                 errors={graphErrors}
    //                 handleCancel={handleCancel}
    //                 handleSubmit={onSubmit}
    //                 isCreate={isCreate}
    //                 isEditing={isEditing}
    //                 loading={isLoading}
    //             />
    //         </Box>
    //     </MaybeLargeDialog>
    // );
    return null;
}

export function RoutineMultiStepCrud({
    // display,
    // isCreate,
    // isOpen,
    // onClose,
    // overrideObject,
    // ...props
}: RoutineMultiStepCrudProps) {
    // const session = useContext(SessionContext);
    // const languages = useMemo(() => getUserLanguages(session), [session]);
    // const [, setLocation] = useLocation();

    // const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<RoutineVersion, RoutineVersionShape>({
    //     ...endpointsRoutineVersion.findOne,
    //     disabled: false, // Never a dialog, so always enabled
    //     isCreate,
    //     objectType: "RoutineVersion",
    //     overrideObject,
    //     transform: (data) => Graph.initialize(session, data),
    // });

    // const [isEditing, setIsEditing] = useState<boolean>(true); //TODO

    // const [changedRoutineVersion, setChangedRoutineVersion] = useState<RoutineVersionShape>(existing);
    // const {
    //     changeInternalValue: pushToChangeStack,
    //     undo,
    //     redo,
    //     canUndo,
    //     canRedo,
    //     resetStack,
    //     stackSize,
    // } = useUndoRedo<RoutineVersionShape>({
    //     initialValue: changedRoutineVersion,
    //     onChange: setChangedRoutineVersion,
    //     forceAddToStack: funcTrue, // Disables debounce. TODO only disable when number/order of nodes or links changes
    // });

    // // The routineVersion's status (valid/invalid/incomplete)
    // const [status, setStatus] = useState<StatusMessageArray>({ status: Status.Incomplete, messages: ["Calculating..."] });

    // useHotkeys([
    //     { keys: ["y", "Z"], ctrlKey: true, callback: redo },
    //     { keys: ["z"], ctrlKey: true, callback: undo },
    // ]);

    // const handleLinkOperation = useCallback(function handleLinkCreateCallback(operation: LinkOperation, link: NodeLink) {
    //     switch (operation) {
    //         case "AddNode":
    //             pushToChangeStack(function change(prev) {
    //                 const node = Graph.generate.nodeRoutineList(prev, languages[0]);
    //                 return Graph.operation.insertNodeAlongLink(prev, link, node);
    //             });
    //             break;
    //         default:
    //             //TODO
    //             break;
    //     }
    // }, [languages, pushToChangeStack]);

    // const handleNodeOperation = useCallback(function handleNodeCreateCallback(operation: GraphElementOperation, node: Node) {
    //     pushToChangeStack((prev) => Graph.node.operate(prev, node, operation));
    // }, [pushToChangeStack]);

    // const handleRoutineListItemOperation = useCallback(function handleRoutineListItemOperationCallback(operation: GraphElementOperation, nodeId: string, item: RoutineListItem) {
    //     pushToChangeStack((prev) => Graph.routineListItem.operate(prev, nodeId, item, operation));
    // }, [pushToChangeStack]);

    // const handleAddSwimLane = useCallback(function handleAddSwimLaneCallback() {
    //     pushToChangeStack((prev) => Graph.addSwimLane(session, prev));
    // }, [pushToChangeStack, session]);

    // // const revertChanges = useCallback(function revertChangesCallback() {
    // //     // Helper function to revert changes
    // //     function revert() {
    // //         setCookieAllowFormCache("RoutineVersion", existing.id, false);
    // //         // If adding, go back
    // //         if (isCreate) {
    // //             window.history.back();
    // //         }
    // //         // Otherwise, revert to original object
    // //         else {
    // //             resetStack();
    // //             setIsEditing(false);
    // //         }
    // //     }
    // //     // Confirm if changes have been made
    // //     if (stackSize.current > 1) {
    // //         PubSub.get().publish("alertDialog", {
    // //             messageKey: "UnsavedChangesBeforeCancel",
    // //             buttons: [
    // //                 { labelKey: "Yes", onClick: () => { revert(); } },
    // //                 { labelKey: "No" },
    // //             ],
    // //         });
    // //     } else {
    // //         revert();
    // //     }
    // // }, [existing.id, isCreate, resetStack, stackSize]);

    // // /**
    // //  * If closing with unsaved changes, prompt user to save
    // //  */
    // // const handleClose = useCallback(function handleCloseCallback() {
    // //     if (isEditing) {
    // //         revertChanges();
    // //     } else {
    // //         keepSearchParams(setLocation, []);
    // //         if (isCreate) {
    // //             window.history.back();
    // //         } else {
    // //             tryOnClose(onClose, setLocation);
    // //         }
    // //     }
    // // }, [isEditing, revertChanges, setLocation, isCreate, onClose]);

    // const hasRoutineChanged = useMemo(() => !isEqual(existing, changedRoutineVersion), [existing, changedRoutineVersion]);

    // const graphErrors = useMemo<FormErrors>(function graphErrorsMemo() {
    //     return {
    //         "graph": status.status === Status.Invalid ? status.messages : null,
    //         "unchanged": !hasRoutineChanged ? "No changes made" : null,
    //     };
    // }, [hasRoutineChanged, status]);

    // async function validateValues(values: RoutineVersionShape) {
    //     return await validateFormValues(values, existing, isCreate, transformRoutineVersionValues, routineVersionValidation);
    // }

    // return (
    //     <Formik
    //         enableReinitialize={true}
    //         initialValues={changedRoutineVersion}
    //         onSubmit={noopSubmit}
    //         validate={validateValues}
    //     >
    //         {(formik) => <RoutineMultiStepCrudForm
    //             disabled={!(isCreate || permissions.canUpdate)}
    //             display={display}
    //             existing={existing}
    //             graphErrors={graphErrors}
    //             handleAddSwimLane={handleAddSwimLane}
    //             handleLinkOperation={handleLinkOperation}
    //             handleNodeOperation={handleNodeOperation}
    //             handleRoutineListItemOperation={handleRoutineListItemOperation}
    //             handleUpdate={setExisting}
    //             hasRoutineChanged={hasRoutineChanged}
    //             isCreate={isCreate}
    //             isEditing={isEditing}
    //             isReadLoading={isReadLoading}
    //             isOpen={isOpen}
    //             status={status}
    //             {...props}
    //             {...formik}
    //         />}
    //     </Formik>
    // );
    return null;
}
