import { HttpStatus, MB_10_BYTES, SERVER_VERSION, ServerError, SessionUser, decodeValue, endpointsApi, endpointsApiKey } from "@local/shared";
import Busboy from "busboy";
import { Express, NextFunction, Request, Response, Router } from "express";
import { SessionService } from "../auth/session";
import { ApiEndpointInfo } from "../builders/types";
import { CustomError } from "../events/error";
import { Context, context } from "../middleware";
import { processAndStoreFiles } from "../utils/fileStorage";
import { ResponseService } from "../utils/response";
import { apiKey_create, apiKey_deleteOne, apiKey_update, apiKey_validate, api_create, api_findMany, api_findOne, api_update } from "./generated";
import { ApiEndpoints } from "./logic/api";
import { ApiKeyEndpoints } from "./logic/apiKey";

const DEFAULT_MAX_FILES = 1;
const DEFAULT_MAX_FILE_SIZE = MB_10_BYTES;

export type EndpointDef = {
    endpoint: string; // e.g. "/bookmark/:id"
    method: "GET" | "POST" | "PUT" | "DELETE";
}
export type EndpointFunction = (
    parent: undefined,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    data: any,
    context: Context,
    info: ApiEndpointInfo,
) => Promise<unknown>;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type FileConfig<TInput = any> = {
    readonly allowedExtensions?: Array<"txt" | "pdf" | "doc" | "docx" | "xls" | "xlsx" | "ppt" | "pptx" | "heic" | "heif" | "png" | "jpg" | "jpeg" | "gif" | "webp" | "tiff" | "bmp" | string>;
    fieldName: string;
    /** Creates the base name for the file */
    fileNameBase?: (input: TInput, currentUser: SessionUser) => string;
    /** Max file size, in bytes */
    maxFileSize?: number;
    maxFiles?: number;
    /** For image files, what they should be resized to */
    imageSizes?: { width: number; height: number }[];
}
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type UploadConfig<TInput = any> = {
    acceptsFiles?: boolean;
    fields: FileConfig<TInput>[];
}
export type EndpointTuple = readonly [
    EndpointDef,
    EndpointFunction,
    ApiEndpointInfo | Record<string, unknown>,
    UploadConfig?
];
export type EndpointType = "get" | "post" | "put" | "delete";
export type EndpointGroup = { [key in EndpointType]?: EndpointTuple };

/**
 * Converts request search params to their correct types. 
 * For example, "true" becomes true, ""true"" becomes "true", and "1" becomes 1.
 * @param input Search parameters from request
 * @returns Object with key/value pairs, or empty object if no params
 */
function parseInput(input: Record<string, unknown>): Record<string, unknown> {
    const parsed: Record<string, unknown> = {};
    Object.entries(input).forEach(([key, value]) => {
        try {
            parsed[key] = decodeValue(JSON.parse(value as string));
        } catch (error) {
            parsed[key] = value;
        }
    });
    return parsed;
}

async function handleEndpoint(
    endpoint: EndpointFunction,
    selection: ApiEndpointInfo,
    input: unknown,
    req: Request,
    res: Response,
) {
    try {
        const data = await endpoint(
            undefined,
            (input ? { input } : undefined),
            context({ req, res }),
            selection,
        );
        ResponseService.sendSuccess(res, data);
    } catch (error: unknown) {
        let formattedError: ServerError;

        if (error instanceof CustomError) {
            formattedError = error.toServerError();
        } else if ((error as Error).name === "ValidationError") {
            formattedError = { trace: "0455", code: "ValidationFailed" };
        } else {
            formattedError = { trace: "0460", code: "InternalError" };
        }

        ResponseService.sendError(res, formattedError, HttpStatus.InternalServerError);
    }
}

/**
 * Middleware to conditionally setup middleware for file uploads.
 */
function maybeFileUploads(config?: UploadConfig) {
    // Return fily upload middleware if the endpoint accepts files.
    return (req: Request, res: Response, next: NextFunction) => {
        if (!config || !config.acceptsFiles) {
            return next();  // No file uploading needed, proceed to next middleware.
        }

        // Find the combined max file size and max number of files, since we don't know which 
        // fields the files are associated with yet.
        // Later on we'll do these checks again, but for each field.
        const totalMaxFileSize = config.fields
            .map(field => field.maxFileSize ?? DEFAULT_MAX_FILE_SIZE)
            .reduce((acc, value) => acc + value, 0);
        const totalMaxFiles = config.fields
            .map(field => field.maxFiles ?? DEFAULT_MAX_FILES)
            .reduce((acc, value) => acc + value, 0);

        const busboy = Busboy({
            headers: req.headers,
            limits: {
                fileSize: totalMaxFileSize,
                files: totalMaxFiles,
            },
        });

        req.files = [];
        req.body = {};

        busboy.on("file", (fieldname, file, info) => {
            const { filename, encoding, mimeType } = info;
            const buffers: Buffer[] = [];

            file.on("data", (data) => buffers.push(data));

            file.on("end", () => {
                if (!req.files) {
                    req.files = [];
                }
                const buffer = Buffer.concat(buffers);
                req.files.push({
                    fieldname,
                    originalname: filename,
                    encoding,
                    mimetype: mimeType,
                    buffer,
                    size: buffer.length,
                });
            });
        });

        busboy.on("field", (fieldname, value) => {
            req.body[fieldname] = value;
        });

        busboy.on("finish", () => next());

        req.pipe(busboy);
    };
}

/**
 * Creates router with endpoints from given object.
 * @param restEndpoints List of endpoint tuples, where each tuple contains 
 * the information needed to create the endpoint.
 */
function setupRoutes(endpointTuples: EndpointTuple[]): Router {
    const router = Router();
    // Loop through each endpoint
    endpointTuples.forEach(([endpointDef, endpointHandler, selection, uploadConfig]) => {
        const { endpoint, method } = endpointDef;
        const methodInLowerCase = method.toLowerCase() as keyof Router;

        // e.g. router.route("/bookmark/:id").get(...)
        const routeChain = router.route(endpoint);

        async function handleEndpointHelper(req: Request, res: Response) {
            // Find non-file data
            let input: Record<string, unknown> | unknown[] = {}; // default to an empty object
            // If it's a GET method, combine params and parsed query
            if (methodInLowerCase === "get") {
                input = Object.assign({}, req.params, parseInput(req.query));
            } else {
                // For non-GET methods, handle object and array bodies
                if (Array.isArray(req.body)) {
                    // Use slice to create a shallow copy of the array
                    input = req.body.slice();
                } else if (typeof req.body === "object" && req.body !== null) {
                    input = Object.assign({}, req.params, req.body);
                }
            }
            // Get files from request
            const files = req.files || [];
            let fileNames: { [x: string]: string[] } = {};
            // Upload files if necessary
            const canUploadFiles =
                uploadConfig?.acceptsFiles
                && Array.isArray(files)
                && files.length > 0
                && (methodInLowerCase === "post" || methodInLowerCase === "put");
            if (canUploadFiles) {
                try {
                    // Must be logged in to upload files
                    const userData = SessionService.getUser(req.session);
                    if (!userData) {
                        ResponseService.sendError(res, { trace: "0509", code: "NotLoggedIn" }, HttpStatus.Unauthorized);
                        return;
                    }
                    fileNames = await processAndStoreFiles(files, input, userData, uploadConfig);
                } catch (error) {
                    ResponseService.sendError(res, { trace: "0506", code: "InternalError" }, HttpStatus.InternalServerError);
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
                    input[fieldname] = { sizes, file };
                }
            }
            handleEndpoint(endpointHandler, selection as ApiEndpointInfo, input, req, res);
        }

        // Add method to route
        routeChain[methodInLowerCase](
            maybeFileUploads(uploadConfig),
            handleEndpointHelper,
        );
    });
    return router;
}

/**
 * Creates a router with all the API endpoints.
 */
export function initRestApi(app: Express) {
    const routes = setupRoutes([
        // API
        [endpointsApi.findOne, ApiEndpoints.Query.api, api_findOne],
        [endpointsApi.findMany, ApiEndpoints.Query.apis, api_findMany],
        [endpointsApi.createOne, ApiEndpoints.Mutation.apiCreate, api_create],
        [endpointsApi.updateOne, ApiEndpoints.Mutation.apiUpdate, api_update],
        // API key
        [endpointsApiKey.createOne, ApiKeyEndpoints.Mutation.apiKeyCreate, apiKey_create],
        [endpointsApiKey.updateOne, ApiKeyEndpoints.Mutation.apiKeyUpdate, apiKey_update],
        [endpointsApiKey.deleteOne, ApiKeyEndpoints.Mutation.apiKeyDeleteOne, apiKey_deleteOne],
        [endpointsApiKey.validateOne, ApiKeyEndpoints.Mutation.apiKeyValidate, apiKey_validate],
    ]);
    app.use(`/api/${SERVER_VERSION}/rest`, routes);
}

export const ApiRest = setupRoutes([

]);
