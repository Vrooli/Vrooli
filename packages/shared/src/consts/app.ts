import { ValueOf } from '.';

export const APP_LINKS = {
    Develop: '/develop', // Develop dashboard
    ForgotPassword: '/forgot-password', // Page for sending password reset request emails
    Home: '/', // Main dashboard for logged in users
    Learn: '/learn', // Learn dashboard
    Orchestration: '/orchestration', // View or update routine orchestration
    Organization: '/organization', // View or update specific organization
    Profile: '/profile', // View or update profile and settings (or view another actor's profile)
    Project: '/project', // View or update specific project
    Projects: '/projects', // View your created and saved projects
    Research: '/research', // Research dashboard
    Routine: '/routine', // View or update specific routine
    Run: '/run', // Displays a UI corresponding to the current subroutine
    ResetPassword: '/password-reset', // Page to reset password, after clicking on password reset link in email
    Search: '/search', // Search routines and users
    Start: '/start', // Provides options for entering application
    Stats: '/stats', // Provides statistics for the website (no admin, so only place to see users, metrics, etc.)
}
export type APP_LINKS = ValueOf<typeof APP_LINKS>;