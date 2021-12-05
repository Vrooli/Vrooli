/* eslint-disable @typescript-eslint/no-redeclare */
import { COOKIE, ValueOf } from '@local/shared';

export const PUBS = {
    ...COOKIE,
    Loading: "loading",
    AlertDialog: "alertDialog",
    Snack: "snack",
    BurgerMenuOpen: "burgerMenuOpen",
    ArrowMenuOpen: "arrowMenuOpen",
    Business: "business",
    Theme: "theme",
}
export type PUBS = ValueOf<typeof PUBS>;

export const FORMS = {
    ForgotPassword: 'forgot-password',
    LogIn: 'login',
    Profile: 'profile',
    ResetPassword: 'reset-password',
    SignUp: 'signup',
}
export type FORMS = ValueOf<typeof FORMS>;

export const LINKS = {
    About: '/about', // Overview of project, the vision, and the team
    Benefits: '/#understand-your-workflow', // Start of slides overviewing benefits of using Vrooli
    Develop: '/develop', // Develop dashboard
    ForgotPassword: '/forgot-password', // Page for sending password reset request emails
    Home: '/app', // Main dashboard for logged in users
    Landing: '/', // Default page when not logged in. Similar to the about page, but more project details and less vision
    Learn: '/learn', // Learn dashboard
    Mission: '/mission', // More details about the project's overall vision
    Orchestration: '/orchestration', // View or update routine orchestration
    Organization: '/organization', // View or update specific organization
    PrivacyPolicy: '/privacy-policy', // Privacy policy
    Profile: '/profile', // View or update profile and settings (or view another actor's profile)
    Project: '/project', // View or update specific project
    Projects: '/projects', // View your created and saved projects
    Research: '/research', // Research dashboard
    Roadmap: '/mission#roadmap', // Start of roadmap slide
    Routine: '/routine', // View or update specific routine
    Run: '/run', // Displays a UI corresponding to the current subroutine
    ResetPassword: '/password-reset', // Page to reset password, after clicking on password reset link in email
    Search: '/search', // Search routines and users
    Start: '/start', // Provides options for entering application
    Stats: '/stats', // Provides statistics for the website (no admin, so only place to see users, metrics, etc.)
    Terms: '/terms-and-conditions', // Terms and conditions
}
export type LINKS = ValueOf<typeof LINKS>;