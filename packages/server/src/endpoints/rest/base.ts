import { Request, Response, Router } from "express";
import { GraphQLResolveInfo } from "graphql";
import i18next from "i18next";
import { getUser } from "../../auth";
import { PartialGraphQLInfo } from "../../builders/types";
import { context, Context } from "../../middleware";
import { GQLEndpoint, IWrap } from "../../types";

export type EndpointFunction<TInput extends object | undefined, TResult extends object> = (
    parent: undefined,
    data: TInput extends undefined ? undefined : IWrap<TInput>,
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

/**
 * Converts request search params to their correct types. 
 * For example, "true" becomes true, ""true"" becomes "true", and "1" becomes 1.
 * @param input Search parameters from request
 * @returns Object with key/value pairs, or empty object if no params
 */
const parseInput = (input: any): Record<string, any> => {
    const parsed: any = {};
    Object.entries(input).forEach(([key, value]) => {
        try {
            parsed[key] = JSON.parse(value as any);
        } catch (error) {
            parsed[key] = value;
        }
    });
    return parsed;
};

const version = "v2";
export const handleEndpoint = async <TInput extends object | undefined, TResult extends object>(
    endpoint: EndpointFunction<TInput, TResult>,
    selection: GraphQLResolveInfo | PartialGraphQLInfo,
    input: TInput | undefined,
    req: Request,
    res: Response,
) => {
    try {
        const data = await endpoint(undefined, (input ? { input } : undefined) as any, context({ req, res }), selection);
        res.json({ data, version });
    } catch (error: any) {
        // Assume that error is from CustomError by default
        const code = error.extensions?.code;
        let message = error.message ?? error.name ?? "";
        // If error is named ValidationError, it's from yup
        if (error.name === "ValidationError") {
            const languages = getUser(req.session)?.languages ?? ["en"];
            const lng = languages.length > 0 ? languages[0] : "en";
            message = i18next.t("error:ValidationFailed", { lng, defaultValue: "Validation failed." });
        }
        // Handle other errors here if needed
        // ...
        res.status(500).json({ errors: [{ code, message }], version });
    }
};

/**
 * Creates router with endpoints from given object.
 * @param restEndpoints Object with endpoints. Each endpoint is an object with 
 * methods as keys and tuples as values. Each tuple has the endpoint function as
 * the first value and the selection as the second value.
 */
export const setupRoutes = (restEndpoints: Record<string, EndpointGroup>) => {
    const router = Router();
    // Loop through each endpoint
    Object.entries(restEndpoints).forEach(([route, methods]) => {
        // Create route
        const routerChain = router.route(route);
        // Loop through each method
        Object.entries(methods).forEach(([method, [endpoint, selection]]) => {
            routerChain[method]((req: Request, res: Response) => {
                // Find input from request
                const input: Record<string, string> = method === "get" ?
                    { ...req.params, ...parseInput(req.query) } :
                    { ...req.params, ...(typeof req.body === "object" ? req.body : {}) };
                handleEndpoint(endpoint as any, selection, input, req, res);
            });
        });
    });
    return router;
};
