import express, { Express, Request, Response } from "express";
import { logger } from "../events";

const canUseValyxa = () => Boolean(process.env.VALYXA_API_KEY);

/**
 * Sets up Valyxa-related routes on the provided Express application instance.
 *
 * @param app - The Express application instance to attach routes to.
 */
export const setupValyxa = (app: Express): void => {
    if (!canUseValyxa()) {
        logger.error("Missing Valyxa API key", { trace: "0490" });
        return;
    }
    // Create webhook endpoint for Valyxa
    app.post("/webhooks/valyxa", express.raw({ type: "application/json" }), async (req: Request, res: Response) => {
        //TODO
    });
};
