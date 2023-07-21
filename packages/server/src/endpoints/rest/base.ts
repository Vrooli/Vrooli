import { NextFunction, Request, Response, Router } from "express";
import { GraphQLResolveInfo } from "graphql";
import i18next from "i18next";
import multer, { Options as MulterOptions } from "multer";
import { getUser } from "../../auth";
import { PartialGraphQLInfo } from "../../builders/types";
import { logger } from "../../events";
import { context, Context } from "../../middleware";
import { GQLEndpoint, IWrap } from "../../types";
import { processAndStoreFiles } from "../../utils";

export type EndpointFunction<TInput extends object | undefined, TResult extends object> = (
    parent: undefined,
    data: TInput extends undefined ? undefined : IWrap<TInput>,
    context: Context,
    info: GraphQLResolveInfo | PartialGraphQLInfo,
) => Promise<TResult>;
export type UploadConfig = {
    acceptsFiles?: boolean;
    fileTypes?: string[];
    maxFileSize?: number;
    maxFiles?: number;
    /** For image files, what they should be resized to */
    imageSizes?: { width: number; height: number }[];
}
export type EndpointTuple = readonly [
    GQLEndpoint<any, any> | GQLEndpoint<never, any>,
    any,
    UploadConfig?
];
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
 * Middleware to conditionally setup multer for file uploads.
 */
export const maybeMulter = (config?: UploadConfig) => {
    // Return multer middleware if the endpoint accepts files.
    return (req: Request, res: Response, next: NextFunction) => {
        if (!config || !config.acceptsFiles) {
            return next();  // No multer needed, proceed to next middleware.
        }

        const multerOptions: MulterOptions = {
            limits: {
                // Maximum file size in bytes. Defaults to 10MB.
                fileSize: config.maxFileSize ?? 10 * 1024 * 1024,
                // Maximum number of files. Defaults to 1.
                files: config.maxFiles ?? 1,
            },
            fileFilter: (_req, file, cb) => {
                // Default to accepting txt and image files
                if (!config?.fileTypes) {
                    config.fileTypes = ["text/plain", "image/jpeg", "image/png"];
                }
                // Only accept files with the correct MIME type
                if (config?.fileTypes && config.fileTypes.includes(file.mimetype)) {
                    cb(null, true);  // accept file
                } else {
                    cb(null, false);  // reject file
                }
            },
        };

        const upload = multer(multerOptions).any();

        return upload(req, res, next);
    };
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
        Object.entries(methods).forEach(([method, [endpoint, selection, config]]) => {
            // Add method to route
            routerChain[method](
                maybeMulter(config),
                // Handle endpoint
                async (req: Request, res: Response) => {
                    // Get files from request
                    const files = req.files as Express.Multer.File[];
                    let fileNames: { [x: string]: string[] } = {};
                    // If there are files and the method is POST or PUT, upload them to S3
                    if (files && (method === "post" || method === "put")) {
                        try {
                            fileNames = await processAndStoreFiles(files, config);
                        } catch (error) {
                            logger.error("Failed to process and store files", { trace: "0506", error });
                            throw error;
                        }
                    }
                    // Find non-file data
                    // TODO add files to input
                    const input: Record<string, string> = method === "get" ?
                        { ...req.params, ...parseInput(req.query) } :
                        { ...req.params, ...(typeof req.body === "object" ? req.body : {}) };
                    handleEndpoint(endpoint as any, selection, input, req, res);
                });
        });
    });
    return router;
};
