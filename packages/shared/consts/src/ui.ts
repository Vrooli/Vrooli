import { ValueOf } from '.';

export const APP_LINKS = {
    Comment: '/comment',
    Develop: '/develop', // Develop dashboard
    DevelopSearch: '/develop-search', // Search page for develop objects
    Example: '/routine/5f0f8f9b-f8f9-4f9b-8f9b-f8f9b8f9b8f9', // Links to example routine that is designed to showcase the UI. See ID of routine set in init seed file
    FAQ: '/#faq', // FAQ section of home page
    ForgotPassword: '/forgot-password', // Page for sending password reset request emails
    History: '/history', // History section of home page. Displays saved, upvoted, recently viewed, completed, and upcoming routines
    HistorySearch: '/history-search', // Search page for history objects
    Home: '/', // Main dashboard for logged in users
    Learn: '/learn', // Learn dashboard
    Organization: '/organization', // View or update specific organization
    Profile: '/profile', // View profile
    Project: '/project', // View or update specific project
    Report: '/report', // Reports view
    Research: '/research', // Research dashboard
    Routine: '/routine', // Displays a UI corresponding to the current subroutine
    ResetPassword: '/password-reset', // Page to reset password, after clicking on password reset link in email
    Search: '/search', // Search page for all objects
    Settings: '/settings', // View or update profile/settings
    Standard: '/standard', // View or update specific standard
    Start: '/start', // Provides options for entering application
    Stats: '/stats', // Provides statistics for the website (no admin, so only place to see users, metrics, etc.)
    Tag: '/tag',
    Tutorial: '/tutorial', // Tutorial for using the application
    User: '/user',
    Welcome: '/welcome', // Displays the first time you enter the application (either as guest or as logged in user)
}
export type APP_LINKS = ValueOf<typeof APP_LINKS>;

export const LANDING_LINKS = {
    AboutUs: '/about',
    Contribute: '/contribute',
    Features: '/features',
    Home: '/',
    PrivacyPolicy: '/privacy-policy', // Privacy policy
    Roadmap: '/mission#roadmap', // Start of roadmap slide
    Terms: '/terms-and-conditions', // Terms and conditions
}
export type LANDING_LINKS = ValueOf<typeof LANDING_LINKS>;

export const THEME = {
    Light: 'light',
    Dark: 'dark'
}
export type THEME = ValueOf<typeof THEME>;