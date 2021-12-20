import jwt from 'jsonwebtoken';
import { Request, Response, NextFunction } from 'express';
import { CODE, COOKIE, ROLES } from '@local/shared';
import { CustomError } from '../error';
import pkg from '@prisma/client';
import { Session, Role } from 'schema/types';
import { RecursivePartial } from '../types';
import { RoleFields } from 'models';
const { PrismaClient } = pkg;

const prisma = new PrismaClient();
const SESSION_MILLI = 30*86400*1000;

// Verifies if a user is authenticated, using an http cookie
export async function authenticate(req: Request, _: Response, next: NextFunction) {
    const { cookies } = req;
    // First, check if a valid session cookie was supplied
    const token = cookies[COOKIE.Session];
    if (token === null || token === undefined) {
        next();
        return;
    }
    if (!process.env.JWT_SECRET) {
        console.error('❗️ JWT_SECRET not set! Please check .env file');
        return;
    }
    // Second, verify that the session token is valid
    jwt.verify(token, process.env.JWT_SECRET, async (error: any, payload: any) => {
        if (error || isNaN(payload.exp) || payload.exp < Date.now()) {
            next();
            return;
        }
        // Now, set token and role variables for other middleware to use
        req.validToken = true;
        req.userId = payload.userId;
        req.roles = payload.roles;
        req.isLoggedIn = payload.isLoggedIn;
        next();
    })
}

interface BasicToken {
    iat: number;
    iss: string;
    exp: number;
}
interface SessionToken extends BasicToken {
    userId?: string;
    roles: RecursivePartial<RoleFields>[];
    isLoggedIn: boolean;
}

/**
 * Generates the minimum data required for a session token
 */
const basicToken = (): BasicToken => ({
    iat: Date.now(),
    iss: `https://app.${process.env.SITE_NAME}/`,
    exp: Date.now() + SESSION_MILLI,

})

/**
 * Generates a JSON Web Token (JWT) for user authentication.
 * The token is added to the "res" object as a cookie.
 * @param res 
 * @param userId 
 * @returns 
 */
export async function generateSessionToken(res: Response, session: RecursivePartial<Session>): Promise<undefined> {
    const tokenContents: SessionToken = {
        ...basicToken(),
        userId: session.id || undefined,
        roles: session.roles as RoleFields[] || [],
        isLoggedIn: Array.isArray(session.roles) ? session.roles.some(r => r?.title === ROLES.Actor) : false,
    }
    if (!process.env.JWT_SECRET) {
        console.error('❗️ JWT_SECRET not set! Please check .env file');
        return;
    }
    const token = jwt.sign(tokenContents, process.env.JWT_SECRET);
    res.cookie(COOKIE.Session, token, {
        httpOnly: true,
        secure: process.env.NODE_ENV === 'production',
        maxAge: SESSION_MILLI
    });
}

// Middleware that restricts access to logged in users
export async function requireLoggedIn(req: any, _: any, next: any) {
    if (!req.isLoggedIn) return new CustomError(CODE.Unauthorized);
    next();
}