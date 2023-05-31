import { Request, Response, Router } from "express";
import { GraphQLResolveInfo } from "graphql";
import { PartialGraphQLInfo } from "../../builders/types";
import { context, Context } from "../../middleware";
import { GQLEndpoint, IWrap } from "../../types";
import { ApiEndpoints } from "../logic";
import { selection } from "./some_path"; // The result selection to import

export type EndpointFunction<TInput, TResult> = (
    parent: undefined,
    data: IWrap<TInput>,
    context: Context,
    info: GraphQLResolveInfo | PartialGraphQLInfo,
) => Promise<TResult>;

const convertQueryToResolveInfo = (query) => {
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
}; // TODO for morning: Move generated gql tags to shared location. Then test convertQueryToResolveInfo and if it works, generate types in script so we don't have to compute them at runtime

// Here's the function that sets up the routes:
function setupRoutes(router: Router, restEndpoints: Record<string, {
    get?: GQLEndpoint<any, any>;
    post?: GQLEndpoint<any, any>;
    put?: GQLEndpoint<any, any>;
    delete?: GQLEndpoint<any, any>;
}>) {
    Object.entries(restEndpoints).forEach(([route, methods]) => {
        const routerChain = router.route(route);
        Object.entries(methods).forEach(([method, endpoint]) => {
            routerChain[method]((req: Request, res: Response) => {
                const input = method === "get" ? { ...req.params, ...req.query } : req.body;
                handleEndpoint(endpoint, input, req, res);
            });
        });
    });
}


export const ApiRest = {
    "/api/:id": {
        get: ApiEndpoints.Query.api,
        put: ApiEndpoints.Mutation.apiUpdate,
    },
    "/apis": {
        get: ApiEndpoints.Query.apis,
    },
    "/api": {
        post: ApiEndpoints.Mutation.apiCreate,
    },
};

const router = Router();
setupRoutes(router, ApiRest);
