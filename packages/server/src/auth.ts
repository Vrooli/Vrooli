import jwt from 'jsonwebtoken';
import { Request, Response, NextFunction } from 'express';
import { CODE, COOKIE } from '@local/shared';
import { CustomError } from './error';
import pkg from '@prisma/client';
const { PrismaClient } = pkg;

const prisma = new PrismaClient();
const SESSION_MILLI = 30*86400*1000;

// Return array of customer roles (ex: ['admin', 'customer'])
const findCustomerRoles = async(customerId: string): Promise<string[]> => {
    // Query customer's roles
    const user = await prisma.customer.findUnique({ 
        where: { id: customerId },
        select: { roles: { select: { role: { select: { title: true } } } } }
    });
    return user?.roles?.map((r: any) => r.role.title.toLowerCase()) || [];
}

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
        req.customerId = payload.customerId;
        req.roles = payload.roles;
        req.isCustomer = payload.isCustomer;
        req.isAdmin = payload.isAdmin;
        next();
    })
}

// Generates a JSON Web Token (JWT)
export async function generateToken(res: Response, customerId: any) {
    const customerRoles = await findCustomerRoles(customerId);
    const tokenContents = {
        iat: Date.now(),
        iss: `https://${process.env.SITE_NAME}/`,
        customerId: customerId,
        roles: customerRoles,
        isCustomer: customerRoles.includes('customer' || 'admin'),
        isAdmin: customerRoles.includes('admin'),
        exp: Date.now() + SESSION_MILLI,
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

// Middleware that restricts access to customers (or admins)
export async function requireCustomer(req: any, _: any, next: any) {
    if (!req.isCustomer) return new CustomError(CODE.Unauthorized);
    next();
}

// Middle ware that restricts access to admins
export async function requireAdmin(req: any, _: any, next: any) {
    if (!req.isAdmin) return new CustomError(CODE.Unauthorized);
    next();
}