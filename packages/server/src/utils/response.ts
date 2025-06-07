import { HttpStatus, SERVER_VERSION, type ServerError, type ServerResponse } from "@vrooli/shared";
import { type Response } from "express";

const FALLBACK_HTTP_STATUS_ERROR = HttpStatus.InternalServerError;

export class ResponseService {
    static sendSuccess<T>(res: Response, data: T, version?: string): void {
        const response: ServerResponse<T> = { data, version };
        res.status(HttpStatus.Ok).json(response);
    }

    static sendError(res: Response, error: ServerError, status = FALLBACK_HTTP_STATUS_ERROR): void {
        const response: ServerResponse = { errors: [error], version: SERVER_VERSION };
        res.status(status).json(response);
    }
}
