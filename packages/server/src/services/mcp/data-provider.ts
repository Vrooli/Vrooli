// A data-provider interface and stub factory for integrating real data services into MCP tools

import { ResourceSubType } from "@local/shared";
import { RequestService } from "../../auth/request.js";
import { resource_findMany } from "../../endpoints/generated/resource_findMany.js";
import { resource } from "../../endpoints/logic/resource.js";
import { getCurrentMcpContext } from "./context.js";
import type { ToolResponse } from "./types.js";

/**
 * Interface defining the data operations for MCP tools.
 */
export interface McpDataProvider {
    /** Fetch one or many resources from the real data layer */
    findResources(args: FindResourcesParams): Promise<ToolResponse>;

    /** Add a new resource or update existing in the real data layer */
    addResource(args: AddResourceParams): Promise<ToolResponse>;

    /** Delete a resource in the real data layer */
    deleteResource(args: DeleteResourceParams): Promise<ToolResponse>;

    /** Update the content of a resource in the real data layer */
    updateResource(args: AddResourceParams): Promise<ToolResponse>;

    /** Execute a routine via the real data layer or side-effects */
    runRoutine(args: RunRoutineParams): Promise<ToolResponse>;
}

/**
 * Factory to create a stubbed McpDataProvider backed by real services.
 * TODO: Replace each method stub with calls to your application's data-access services.
 */
export function createMcpDataProvider(): McpDataProvider {
    return {
        findResources: async function ({ id, query, resource_type }: FindResourcesParams): Promise<ToolResponse> {
            // Retrieve HTTP request context and enforce permissions
            const { req, res } = getCurrentMcpContext();
            RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });

            // Build GraphQL input for resource.findMany
            const input: Record<string, any> = {};
            if (id) {
                input.rootId = id;
            }
            if (query) {
                input.searchString = query;
            }
            // Only filter for routines; other types fetch unfiltered
            if (resource_type === "Routine") {
                input.resourceSubTypes = [ResourceSubType.RoutineMultiStep];
            }

            // Call official resource.findMany endpoint
            const result = await resource.findMany(
                { input },
                { req, res },
                resource_findMany,
            );

            // Wrap in MCP ToolResponse
            return { content: [{ type: 'text', text: JSON.stringify(result, null, 2) }] };
        },
        addResource: async (args) => {
            throw new Error("addResource not implemented");
        },
        deleteResource: async (args) => {
            throw new Error("deleteResource not implemented");
        },
        updateResource: async (args) => {
            throw new Error("updateResource not implemented");
        },
        runRoutine: async (args) => {
            throw new Error("runRoutine not implemented");
        },
    };
} 