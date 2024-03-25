import { Request, Response } from "express";

interface ContextProps {
    req: Request;
    res: Response;
}
export type Context = ContextProps;
export const context = ({ req, res }: ContextProps): Context => ({
    req,
    res,
});
