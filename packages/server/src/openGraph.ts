import { Request, Response, NextFunction } from 'express';

/**
 * Regex to test if a user agent is an open graph crawler. Make sure 
 * not to include googlebot, baiduspider, or any other search engine
 * crawlers here.
 */
const openGraphRegex = /facebookexternalhit|twitterbot|linkedinbot|embedly|quora link preview|Pinterest|slackbot|vkShare|facebot/i;

/**
 * Determines if a user agent is a bot (sprcifically, an open graph crawler) or not.
 * @param req The request object
 * @returns True if the user agent is a bot, false otherwise
 */
export function isBot(req: Request): boolean {
    console.log('checking user agent', req.headers['user-agent']);
    const userAgent = req.headers['user-agent'];
    if (!userAgent) return false;
    return openGraphRegex.test(userAgent);
}

/**
 * Middleware to send Open Graph data to social sharing bots
 * @param req The request object
 * @param res The response object
 * @param next The next function to call.
 */
export async function openGraphChecker(req: Request, res: Response, next: NextFunction) {
    if (isBot(req)) {
        console.log('bot detected');
    }
    next();
}