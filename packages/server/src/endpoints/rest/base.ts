import { HttpStatus, decodeValue } from "@local/shared";
import { NextFunction, Request, Response, Router } from "express";
import { GraphQLResolveInfo } from "graphql";
import i18next from "i18next";
import multer, { Options as MulterOptions } from "multer";
import { getUser } from "../../auth/request";
import { PartialGraphQLInfo } from "../../builders/types";
import { CustomError } from "../../events/error";
import { Context, context } from "../../middleware";
import { GQLEndpoint, IWrap, SessionUserToken } from "../../types";
import { processAndStoreFiles } from "../../utils/fileStorage";

export type EndpointFunction<TInput extends object | undefined, TResult extends object> = (
    parent: undefined,
    data: TInput extends undefined ? undefined : IWrap<TInput>,
    context: Context,
    info: GraphQLResolveInfo | PartialGraphQLInfo,
) => Promise<TResult>;
export type FileConfig<TInput extends object | undefined> = {
    readonly allowedExtensions?: Array<"txt" | "pdf" | "doc" | "docx" | "xls" | "xlsx" | "ppt" | "pptx" | "heic" | "heif" | "png" | "jpg" | "jpeg" | "gif" | "webp" | "tiff" | "bmp" | string>;
    fieldName: string;
    /** Creates the base name for the file */
    fileNameBase?: (input: TInput, currentUser: SessionUserToken) => string;
    /** Max file size, in bytes */
    maxFileSize?: number;
    maxFiles?: number;
    /** For image files, what they should be resized to */
    imageSizes?: { width: number; height: number }[];
}
export type UploadConfig<TInput extends object | undefined> = {
    acceptsFiles?: boolean;
    fields: FileConfig<TInput>[];
}
export type EndpointTuple = readonly [
    GQLEndpoint<any, any> | GQLEndpoint<never, any>,
    any,
    UploadConfig<any>?
];
export type EndpointType = "get" | "post" | "put" | "delete";
export type EndpointGroup = { [key in EndpointType]?: EndpointTuple };

/**
 * Converts request search params to their correct types. 
 * For example, "true" becomes true, ""true"" becomes "true", and "1" becomes 1.
 * @param input Search parameters from request
 * @returns Object with key/value pairs, or empty object if no params
 */
function parseInput(input: any): Record<string, any> {
    const parsed: any = {};
    Object.entries(input).forEach(([key, value]) => {
        try {
            parsed[key] = decodeValue(JSON.parse(value as any));
        } catch (error) {
            parsed[key] = value;
        }
    });
    return parsed;
}

const version = "v2";
export async function handleEndpoint<TInput extends object | undefined, TResult extends object>(
    endpoint: EndpointFunction<TInput, TResult>,
    selection: GraphQLResolveInfo | PartialGraphQLInfo,
    input: TInput | undefined,
    req: Request,
    res: Response,
) {
    try {
        const data = await Promise.resolve(endpoint(undefined, (input ? { input } : undefined) as any, context({ req, res }), selection));
        res.json({ data, version });
    } catch (error: unknown) {
        let code: string | undefined;
        let message: string;

        if (error instanceof CustomError) {
            // Use the code and message directly from CustomError
            code = error.code;
            message = error.message;
        } else if ((error as Error).name === "ValidationError") {
            // Handle ValidationError from Yup
            const languages = getUser(req.session)?.languages ?? ["en"];
            const lng = languages.length > 0 ? languages[0] : "en";
            message = i18next.t("error:ValidationFailed", {
                lng,
                defaultValue: "Validation failed.",
            });
            code = "ValidationError";
        } else {
            // Handle unexpected errors
            message = "An unexpected error occurred.";
            code = "InternalServerError";
        }

        res.status(HttpStatus.InternalServerError).json({
            errors: [{ code, message }],
            version,
        });
    }
}

/**
 * Middleware to conditionally setup multer for file uploads.
 */
export function maybeMulter<TInput extends object | undefined>(config?: UploadConfig<TInput>) {
    // Return multer middleware if the endpoint accepts files.
    return (req: Request, res: Response, next: NextFunction) => {
        if (!config || !config.acceptsFiles) {
            return next();  // No multer needed, proceed to next middleware.
        }

        // Find the combined max file size and max number of files, since we don't know which 
        // fields the files are associated with yet.
        // Later on we'll do these checks again, but for each field.
        const totalMaxFileSize = config.fields
            .map(field => field.maxFileSize ?? (10 * 1024 * 1024)) // Each field defaults to 10MB if not specified.
            .reduce((acc, value) => acc + value, 0);
        const totalMaxFiles = config.fields
            .map(field => field.maxFiles ?? 1) // Each field defaults to 1 if not specified.
            .reduce((acc, value) => acc + value, 0);
        // Create multer options
        const multerOptions: MulterOptions = {
            limits: {
                // Maximum file size in bytes. Defaults to 10MB.
                fileSize: totalMaxFileSize,
                // Maximum number of files. Defaults to 1.
                files: totalMaxFiles,
            },
        };

        const upload = multer(multerOptions).any();

        return upload(req, res, next);
    };
}

/**
 * Creates router with endpoints from given object.
 * @param restEndpoints Object with endpoints. Each endpoint is an object with 
 * methods as keys and tuples as values. Each tuple has the endpoint function as
 * the first value and the selection as the second value.
 */
export function setupRoutes(restEndpoints: Record<string, EndpointGroup>) {
    const router = Router();
    // Loop through each endpoint
    Object.entries(restEndpoints).forEach(([route, methods]) => {
        // Create route
        const routerChain = router.route(route);
        // Loop through each method
        (Object.entries(methods) as [EndpointType, EndpointTuple][]).forEach(([method, [endpoint, selection, config]]) => {
            // Add method to route
            routerChain[method](
                maybeMulter(config),
                // Handle endpoint
                async (req: Request, res: Response) => {
                    // Find non-file data
                    let input: Record<string, string> | any[] = {}; // default to an empty object
                    // If it's a GET method, combine params and parsed query
                    if (method === "get") {
                        input = { ...req.params, ...parseInput(req.query) };
                    } else {
                        // For non-GET methods, handle object and array bodies
                        if (Array.isArray(req.body)) {
                            input = [...req.body]; // directly spread array into a new array
                        } else if (typeof req.body === "object") {
                            input = { ...req.params, ...req.body };
                        }
                    }
                    // Get files from request
                    const files = (req.files ?? []) as Express.Multer.File[];
                    let fileNames: { [x: string]: string[] } = {};
                    // If there are files and the method is POST or PUT, upload them to S3
                    if (Array.isArray(files) && files.length > 0 && (method === "post" || method === "put")) {
                        try {
                            // Must be logged in to upload files
                            const userData = getUser(req.session);
                            if (!userData) {
                                const languages = getUser(req.session)?.languages ?? ["en"];
                                const lng = languages.length > 0 ? languages[0] : "en";
                                res.status(HttpStatus.Unauthorized).json({ errors: [{ code: "0509", message: i18next.t("error:NotLoggedIn", { lng, defaultValue: "Not logged in." }) }], version });
                                return;
                            }
                            fileNames = await processAndStoreFiles(files, input, userData, config);
                        } catch (error) {
                            const languages = getUser(req.session)?.languages ?? ["en"];
                            const lng = languages.length > 0 ? languages[0] : "en";
                            res.status(HttpStatus.InternalServerError).json({ errors: [{ code: "0506", message: i18next.t("error:InternalError", { lng, defaultValue: "Internal error." }) }], version });
                            return;
                        }
                    }
                    // Add files to input
                    for (const { fieldname, originalname } of files) {
                        // If there are no files, upload must have failed or been rejected. So skip this file.
                        const currFileNames = fileNames[originalname];
                        if (!currFileNames || currFileNames.length === 0) continue;
                        // We're going to store image files differently than other files. 
                        // For normal files, there should only be one file in the array. So we'll just use that.
                        if (currFileNames.length === 1) {
                            input[fieldname] = currFileNames[0] as string;
                        }
                        // For image files, there may be multiple sizes. We'll store them like this: 
                        // { sizes: ["1024x1024", "512x512"], file: "https://bucket-name.s3.region.amazonaws.com/image-hash_*.jpg }
                        // Note how there is an asterisk in the file name. You can replace this with any of the sizes to create a valid URL.
                        else {
                            // Find the size string of each file
                            const sizes = currFileNames.map((file: string) => {
                                const url = new URL(file);
                                const size = url.pathname.split("/").pop()?.split("_")[1]?.split(".")[0];
                                return size;
                            }).filter((size: string | undefined) => size !== undefined) as string[];
                            // Find the file name without the size string
                            const file = (currFileNames[0] as string).replace(`_${sizes[0]}`, "_*");
                            input[fieldname] = JSON.stringify({ sizes, file });
                        }
                    }
                    handleEndpoint(endpoint as any, selection, input, req, res);
                });
        });
    });
    return router;
}
