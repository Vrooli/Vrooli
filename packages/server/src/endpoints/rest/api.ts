import { apiCreate, apiFindMany, apiFindOne, apiUpdate } from "@local/shared";
import { Request, Response, Router } from "express";
import { DocumentNode, FieldNode, FragmentDefinitionNode, GraphQLResolveInfo, OperationDefinitionNode } from "graphql";
import { PartialGraphQLInfo } from "../../builders/types";
import { context, Context } from "../../middleware";
import { GQLEndpoint, IWrap } from "../../types";
import { ApiEndpoints } from "../logic";

export type EndpointFunction<TInput, TResult> = (
    parent: undefined,
    data: IWrap<TInput>,
    context: Context,
    info: GraphQLResolveInfo | PartialGraphQLInfo,
) => Promise<TResult>;

export type EndpointTuple = readonly [GQLEndpoint<any, any>, GraphQLResolveInfo | PartialGraphQLInfo];

const gqlToGraphQLResolveInfo = (query: DocumentNode, path: string): GraphQLResolveInfo => {
    const operation: OperationDefinitionNode | undefined = query.definitions
        .find(({ kind }) => kind === "OperationDefinition") as OperationDefinitionNode;

    const fragmentDefinitions = query.definitions
        .filter(({ kind }) => kind === "FragmentDefinition") as FragmentDefinitionNode[];

    const fragments: { [key: string]: FragmentDefinitionNode } = fragmentDefinitions
        .reduce((result, current: FragmentDefinitionNode) => ({
            ...result,
            [current.name.value]: current,
        }), {});

    const fieldNodes: FieldNode[] = operation?.selectionSet.selections as FieldNode[];

    const resolveInfo: GraphQLResolveInfo = {
        fieldName: fieldNodes[0]?.name.value || "",
        fieldNodes,
        returnType: null,
        parentType: null,
        schema: null,
        fragments,
        rootValue: {},
        operation,
        variableValues: {},
        path: {
            prev: undefined,
            key: path,
        },
    } as any;
    return resolveInfo;
};

export const handleEndpoint = async <TInput, TResult>(
    endpoint: EndpointFunction<TInput, TResult>,
    selection: GraphQLResolveInfo | PartialGraphQLInfo,
    input: TInput,
    req: Request,
    res: Response,
) => {
    try {
        const result = await endpoint(undefined, { input }, context({ req, res }), selection);
        res.json(result);
    } catch (error: any) {
        res.status(500).json({ error: error.toString() });
    }
};

// Here's the function that sets up the routes:
function setupRoutes(router: Router, restEndpoints: Record<string, {
    get?: EndpointTuple;
    post?: EndpointTuple;
    put?: EndpointTuple;
    delete?: EndpointTuple;
}>) {
    Object.entries(restEndpoints).forEach(([route, methods]) => {
        const routerChain = router.route(route);
        Object.entries(methods).forEach(([method, [endpoint, selection]]) => {
            routerChain[method]((req: Request, res: Response) => {
                const input: PartialGraphQLInfo = method === "get" ? { ...req.params, ...req.query } : req.body;
                handleEndpoint(endpoint as any, selection, input, req, res);
            });
        });
    });
}


export const ApiRest = {
    "/api/:id": {
        get: [ApiEndpoints.Query.api, gqlToGraphQLResolveInfo(apiFindOne, "apiFindOne")],
        put: [ApiEndpoints.Mutation.apiUpdate, gqlToGraphQLResolveInfo(apiUpdate, "apiUpdate")],
    },
    "/apis": {
        get: [ApiEndpoints.Query.apis, gqlToGraphQLResolveInfo(apiFindMany, "apiFindMany")],
    },
    "/api": {
        post: [ApiEndpoints.Mutation.apiCreate, gqlToGraphQLResolveInfo(apiCreate, "apiCreate")],
    },
} as const;

const router = Router();
setupRoutes(router, ApiRest);
export default router;
