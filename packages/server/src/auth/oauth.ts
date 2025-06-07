// import { AUTH_ROUTE_PREFIX, OAUTH_PROVIDERS, getOAuthCallbackRoute, getOAuthInitRoute } from "@vrooli/shared";
// import pkg from "@prisma/client";
// import { Request, Response } from "express";
// import passport from "passport";
// import { Strategy as GoogleStrategy, Profile } from "passport-google-oauth20";
// import { app } from "../app";

// const { PrismaClient } = pkg;

// //TODO need to use this file in server/index.ts

// const prisma = new PrismaClient();

// // Define Strategies for each OAuth provider

// passport.use(
//     new GoogleStrategy(
//         {
//             clientID: process.env.GOOGLE_CLIENT_ID!, //TODO keys not in env file
//             clientSecret: process.env.GOOGLE_CLIENT_SECRET!,
//             callbackURL: "/auth/google/callback",
//             passReqToCallback: true,
//         },
//         async (accessToken: string, refreshToken: string, profile: Profile, done) => {
//             try {
//                 // TODO might need to do jwt token verification here
//                 // Check if user already exists
//                 const existingUser = await prisma.user.findUnique({
//                     where: { googleId: profile.id },
//                 });
//                 if (existingUser) {
//                     return done(null, existingUser);
//                 }
//                 // If not, create a new user
//                 const user = await prisma.user.create({
//                     data: {
//                         googleId: profile.id,
//                         name: profile.displayName,
//                         email: profile.emails?.[0].value,
//                         // Add other fields as necessary
//                     },
//                 });
//                 return done(null, user);
//             } catch (err) {
//                 return done(err);
//             }
//         },
//     ),
// );

// // Initialize Passport middleware
// app.use(passport.initialize());

// // Initiate Google OAuth
// app.get(
//     getOAuthInitRoute(OAUTH_PROVIDERS.Google).replace(AUTH_ROUTE_PREFIX, ""),
//     passport.authenticate(OAUTH_PROVIDERS.Google, { scope: ["profile", "email"] }),
// );

// // Handle Google OAuth callback
// app.get(
//     getOAuthCallbackRoute(OAUTH_PROVIDERS.Google).replace(AUTH_ROUTE_PREFIX, ""),
//     passport.authenticate(OAUTH_PROVIDERS.Google, { failureRedirect: "/login-failure" }),
//     (req: Request, res: Response) => {
//         res.redirect("/login-success");
//     },
// );
export { };

