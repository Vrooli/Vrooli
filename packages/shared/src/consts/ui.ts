import { ValueOf } from '.';

export const APP_LINKS = {
    Build: '/build', // View or update routine orchestration
    Develop: '/develop', // Develop dashboard
    Example: '/build/5f0f8f9b-f8f9-4f9b-8f9b-f8f9b8f9b8f9', // Links to example routine that is designed to showcase the UI. See ID of routine set in init seed file
    FAQ: '/#faq', // FAQ section of home page
    ForgotPassword: '/forgot-password', // Page for sending password reset request emails
    ForYou: '/for-you', // For you section of home page. Displays saved, upvoted, recently viewed, completed, and upcoming routines
    Home: '/', // Main dashboard for logged in users
    Learn: '/learn', // Learn dashboard
    Organization: '/organization', // View or update specific organization
    Profile: '/profile', // View profile
    Project: '/project', // View or update specific project
    Research: '/research', // Research dashboard
    Run: '/run', // Displays a UI corresponding to the current subroutine
    ResetPassword: '/password-reset', // Page to reset password, after clicking on password reset link in email
    SearchOrganizations: '/search/organization', // Search organizations
    SearchProjects: '/search/project', // Search projects
    SearchRoutines: '/search/routine', // Search routines
    SearchStandards: '/search/standard', // Search standards
    SearchUsers: '/search/user', // Search users
    Settings: '/settings', // View or update profile/settings
    Standard: '/standard', // View or update specific standard
    Start: '/start', // Provides options for entering application
    Stats: '/stats', // Provides statistics for the website (no admin, so only place to see users, metrics, etc.)
    Welcome: '/welcome', // Displays the first time you enter the application (either as guest or as logged in user)
}
export type APP_LINKS = ValueOf<typeof APP_LINKS>;

export const LANDING_LINKS = {
    About: '/about', // Overview of project, the vision, and the team
    Benefits: '/#understand-your-workflow', // Start of slides overviewing benefits of using Vrooli
    Home: '/', // Default page when not logged in. Similar to the about page, but more project details and less vision
    Mission: '/mission', // More details about the project's overall vision
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