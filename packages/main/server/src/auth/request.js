import { COOKIE } from "@local/consts";
import { uuidValidate } from "@local/uuid";
import fs from "fs";
import jwt from "jsonwebtoken";
import { CustomError } from "../events/error";
import { logger } from "../events/logger";
import { isSafeOrigin } from "../utils";
const SESSION_MILLI = 30 * 86400 * 1000;
const privateKey = fs.readFileSync(`${process.env.PROJECT_DIR}/jwt_priv.pem`, "utf8");
const publicKey = fs.readFileSync(`${process.env.PROJECT_DIR}/jwt_pub.pem`, "utf8");
const parseAcceptLanguage = (req) => {
    const acceptString = req.headers["accept-language"];
    if (!acceptString || acceptString === "*")
        return ["en"];
    let acceptValues = acceptString.split(",").map((lang) => lang.split(";")[0]);
    acceptValues = acceptValues.map((lang) => lang.split("-")[0]);
    return acceptValues;
};
export async function authenticate(req, res, next) {
    try {
        const { cookies } = req;
        req.fromSafeOrigin = isSafeOrigin(req);
        req.languages = parseAcceptLanguage(req);
        const token = cookies[COOKIE.Jwt];
        if (token === null || token === undefined) {
            let error;
            if (!req.fromSafeOrigin)
                error = new CustomError("0247", "UnsafeOriginNoApiToken", req.languages);
            next(error);
            return;
        }
        jwt.verify(token, publicKey, { algorithms: ["RS256"] }, async (error, payload) => {
            try {
                if (error || isNaN(payload.exp) || payload.exp < Date.now()) {
                    let error;
                    if (!req.fromSafeOrigin)
                        error = new CustomError("0248", "UnsafeOriginNoApiToken", req.languages);
                    next(error);
                    return;
                }
                req.apiToken = payload.apiToken ?? false;
                req.isLoggedIn = payload.isLoggedIn === true && Array.isArray(payload.users) && payload.users.length > 0;
                req.timeZone = payload.timeZone ?? "UTC";
                req.users = [...new Map((payload.users ?? []).map((user) => [user.id, user])).values()];
                if (req.users.length && req.users[0].languages && req.users[0].languages.length) {
                    let languages = req.users[0].languages;
                    languages.push(...req.languages);
                    languages = [...new Set(languages)];
                    req.languages = languages;
                }
                req.validToken = true;
                next();
            }
            catch (error) {
                logger.error("Error verifying token", { trace: "0450", error });
                res.clearCookie(COOKIE.Jwt);
                next(error);
            }
        });
    }
    catch (error) {
        logger.error("Error authenticating request", { trace: "0451", error });
        next(error);
    }
}
const basicToken = () => ({
    iat: Date.now(),
    iss: "https://vrooli.com/",
    exp: Date.now() + SESSION_MILLI,
});
export async function generateSessionJwt(res, session) {
    const tokenContents = {
        ...basicToken(),
        isLoggedIn: session.isLoggedIn ?? false,
        timeZone: session.timeZone ?? undefined,
        users: [...new Map((session.users ?? []).map((user) => [user.id, {
                    id: user.id,
                    activeFocusMode: user.activeFocusMode ? {
                        mode: {
                            id: user.activeFocusMode.mode?.id,
                        },
                        stopCondition: user.activeFocusMode.stopCondition,
                        stopTime: user.activeFocusMode.stopTime,
                    } : undefined,
                    handle: user.handle,
                    hasPremium: user.hasPremium ?? false,
                    languages: user.languages ?? [],
                    name: user.name ?? undefined,
                }])).values()],
    };
    const token = jwt.sign(tokenContents, privateKey, { algorithm: "RS256" });
    res.cookie(COOKIE.Jwt, token, {
        httpOnly: true,
        secure: process.env.NODE_ENV === "production",
        maxAge: SESSION_MILLI,
    });
}
export async function generateApiJwt(res, apiToken) {
    const tokenContents = {
        ...basicToken(),
        apiToken,
    };
    const token = jwt.sign(tokenContents, privateKey, { algorithm: "RS256" });
    res.cookie(COOKIE.Jwt, token, {
        httpOnly: true,
        secure: process.env.NODE_ENV === "production",
        maxAge: SESSION_MILLI,
    });
}
export async function updateSessionTimeZone(req, res, timeZone) {
    if (req.timeZone === timeZone)
        return;
    const { cookies } = req;
    const token = cookies[COOKIE.Jwt];
    if (token === null || token === undefined) {
        logger.error("❗️ No session token found", { trace: "0006" });
        return;
    }
    jwt.verify(token, publicKey, { algorithms: ["RS256"] }, async (error, payload) => {
        if (error || isNaN(payload.exp) || payload.exp < Date.now()) {
            logger.error("❗️ Session token is invalid", { trace: "0008" });
            return;
        }
        const tokenContents = {
            ...payload,
            timeZone,
        };
        const newToken = jwt.sign(tokenContents, privateKey, { algorithm: "RS256" });
        res.cookie(COOKIE.Jwt, newToken, {
            httpOnly: true,
            secure: process.env.NODE_ENV === "production",
            maxAge: payload.exp - Date.now(),
        });
    });
}
export async function updateSessionCurrentUser(req, res, user) {
    const { cookies } = req;
    const token = cookies[COOKIE.Jwt];
    if (token === null || token === undefined) {
        logger.error("❗️ No session token found", { trace: "0445" });
        return;
    }
    jwt.verify(token, publicKey, { algorithms: ["RS256"] }, async (error, payload) => {
        if (error || isNaN(payload.exp) || payload.exp < Date.now()) {
            logger.error("❗️ Session token is invalid", { trace: "0447" });
            return;
        }
        const tokenContents = {
            ...payload,
            users: payload.users?.length > 0 ? [{
                    ...payload.users[0], ...{
                        activeFocusMode: user.activeFocusMode ? {
                            mode: {
                                id: user.activeFocusMode.mode?.id,
                            },
                            stopCondition: user.activeFocusMode.stopCondition,
                            stopTime: user.activeFocusMode.stopTime,
                        } : undefined,
                        handle: user.handle,
                        hasPremium: user.hasPremium ?? false,
                        languages: user.languages ?? [],
                        name: user.name ?? undefined,
                    },
                }, ...payload.users.slice(1)] : [],
        };
        const newToken = jwt.sign(tokenContents, privateKey, { algorithm: "RS256" });
        res.cookie(COOKIE.Jwt, newToken, {
            httpOnly: true,
            secure: process.env.NODE_ENV === "production",
            maxAge: payload.exp - Date.now(),
        });
    });
}
export async function requireLoggedIn(req, _, next) {
    let error;
    if (!req.isLoggedIn)
        error = new CustomError("0018", "NotLoggedIn", req.languages);
    next(error);
}
export const getUser = (req) => {
    if (!req || !Array.isArray(req?.users) || req.users.length === 0)
        return null;
    const user = req.users[0];
    return typeof user.id === "string" && uuidValidate(user.id) ? user : null;
};
export const assertRequestFrom = (req, conditions) => {
    const userData = getUser(req);
    const hasUserData = req.isLoggedIn === true && Boolean(userData);
    const hasApiToken = req.apiToken === true;
    if (conditions.isApiRoot !== undefined) {
        const isApiRoot = hasApiToken && !hasUserData;
        if (conditions.isApiRoot === true && !isApiRoot)
            throw new CustomError("0265", "MustUseApiToken", req.languages);
        if (conditions.isApiRoot === false && isApiRoot)
            throw new CustomError("0266", "MustNotUseApiToken", req.languages);
    }
    if (conditions.isUser !== undefined) {
        const isUser = hasUserData && (hasApiToken || req.fromSafeOrigin === true);
        if (conditions.isUser === true && !isUser)
            throw new CustomError("0267", "NotLoggedIn", req.languages);
        if (conditions.isUser === false && isUser)
            throw new CustomError("0268", "NotLoggedIn", req.languages);
    }
    if (conditions.isOfficialUser !== undefined) {
        const isOfficialUser = hasUserData && !hasApiToken && req.fromSafeOrigin === true;
        if (conditions.isOfficialUser === true && !isOfficialUser)
            throw new CustomError("0269", "NotLoggedInOfficial", req.languages);
        if (conditions.isOfficialUser === false && isOfficialUser)
            throw new CustomError("0270", "NotLoggedInOfficial", req.languages);
    }
    return conditions.isUser === true || conditions.isOfficialUser === true ? userData : undefined;
};
//# sourceMappingURL=request.js.map