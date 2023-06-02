import { Request, Response, Router } from "express";
import { GraphQLResolveInfo } from "graphql";
import { PartialGraphQLInfo } from "../../builders/types";
import { context, Context } from "../../middleware";
import { GQLEndpoint, IWrap } from "../../types";

export type EndpointFunction<TInput, TResult> = (
    parent: undefined,
    data: IWrap<TInput>,
    context: Context,
    info: GraphQLResolveInfo | PartialGraphQLInfo,
) => Promise<TResult>;
export type EndpointTuple = readonly [GQLEndpoint<any, any> | GQLEndpoint<never, any>, any];//GraphQLResolveInfo | PartialGraphQLInfo];
export type EndpointGroup = {
    get?: EndpointTuple;
    post?: EndpointTuple;
    put?: EndpointTuple;
    delete?: EndpointTuple;
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

export const setupRoutes = (restEndpoints: Record<string, EndpointGroup>) => {
    const router = Router();
    Object.entries(restEndpoints).forEach(([route, methods]) => {
        const routerChain = router.route(route);
        Object.entries(methods).forEach(([method, [endpoint, selection]]) => {
            routerChain[method]((req: Request, res: Response) => {
                const input: PartialGraphQLInfo = method === "get" ? { ...req.params, ...req.query } : req.body;
                handleEndpoint(endpoint as any, selection, input, req, res);
            });
        });
    });
    return router;
};
