import { ValueOf } from '.';

export const APP_LINKS = {
    Api: '/api',
    Comment: '/comment',
    Create: '/create',
    Example: '/routine/5f0f8f9b-f8f9-4f9b-8f9b-f8f9b8f9b8f9', // Links to example routine that is designed to showcase the UI. See ID of routine set in init seed file
    FAQ: '/#faq', // FAQ section of home page
    ForgotPassword: '/forgot-password',
    History: '/history',
    HistorySearch: '/history-search',
    Home: '/', // Main dashboard for logged in users
    Note: '/note',
    Notifications: '/notifications',
    Organization: '/organization',
    Profile: '/profile',
    Project: '/project',
    Question: '/question',
    Quiz: '/quiz',
    Reminder: '/reminder',
    Report: '/report',
    Routine: '/routine',
    ResetPassword: '/password-reset',
    Search: '/search',
    Settings: '/settings',
    SmartContract: '/contract',
    Standard: '/standard',
    Start: '/start',
    Stats: '/stats', // Provides statistics for the website (no admin, so only place to see users, metrics, etc.)
    Tag: '/tag',
    Tutorial: '/tutorial',
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