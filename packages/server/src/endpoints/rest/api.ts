import { apiCreate, apiFindMany, apiFindOne, apiUpdate } from "@local/shared";
import { Request, Response, Router } from "express";
import { GraphQLResolveInfo } from "graphql";
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

export type EndpointTuple = readonly [GQLEndpoint<any, any>, PartialGraphQLInfo];

const gqlToGraphQLResolveInfo = (query) => {
    const operation = query.definitions
        .find(({ kind }) => kind === "OperationDefinition");
    const fragments = query.definitions
        .filter(({ kind }) => kind === "FragmentDefinition")
        .reduce((result, current) => ({
            ...result,
            [current.name.value]: current,
        }), {});

    return {
        fieldNodes: operation.selectionSet.selections,
        fragments,
    };
};

export const handleEndpoint = async <TInput, TResult>(
    endpoint: EndpointFunction<TInput, TResult>,
    selection: PartialGraphQLInfo,
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
        get: [ApiEndpoints.Query.api, gqlToGraphQLResolveInfo(apiFindOne)],
        put: [ApiEndpoints.Mutation.apiUpdate, gqlToGraphQLResolveInfo(apiUpdate)],
    },
    "/apis": {
        get: [ApiEndpoints.Query.apis, gqlToGraphQLResolveInfo(apiFindMany)],
    },
    "/api": {
        post: [ApiEndpoints.Mutation.apiCreate, gqlToGraphQLResolveInfo(apiCreate)],
    },
} as const;

const router = Router();
setupRoutes(router, ApiRest);
