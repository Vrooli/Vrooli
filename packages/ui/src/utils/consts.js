import { COOKIE } from '@local/shared';

export const SORT_OPTIONS = [
    {
        label: 'A-Z',
        value: 'AZ',
    },
    {
        label: 'Z-A',
        value: 'ZA',
    },
    {
        label: 'Price: Low to High',
        value: 'PriceLowHigh',
    },
    {
        label: 'Price: High to Low',
        value: 'PriceHighLow',
    },
    {
        label: 'Featured',
        value: 'Featured',
    },
    {
        label: 'Newest',
        value: 'Newest',
    },
    {
        label: 'Oldest',
        value: 'Oldest',
    }
]

export const PUBS = {
    ...COOKIE,
    Loading: "loading",
    AlertDialog: "alertDialog",
    Snack: "snack",
    BurgerMenuOpen: "burgerMenuOpen",
    ArrowMenuOpen: "arrowMenuOpen",
    Business: "business",
}

export const LINKS = {
    About: '/about', // Overview of project, the vision, and the team
    Admin: '/admin', // Admin portal. Contains cards that link to every admin page
    AdminContactInfo: '/admin/contact-info', // Admin page for updating contact information (business.json file)
    AdminCustomers: '/admin/customers', // Admin page for customer statistics and contact information
    AdminHero: '/admin/hero', // Admin page for updating hero slideshow
    ForgotPassword: '/forgot-password', // Page for sending password reset request emails
    Home: '/', // Home page. Similar to the about page, but more project details and less vision
    LogIn: '/login', // Page with log in form
    Portal: '/portal', // Main page for logged-in customers. Shows created and liked routines, displayed either as a knowledge graph or classical file structure
    PrivacyPolicy: '/privacy-policy', // Privacy policy
    Profile: '/profile', // View and update profile and settings
    Register: '/register', // Page with register form
    Run: '/run', // Displays a UI corresponding to the current subroutine
    ResetPassword: '/password-reset', // Page to reset password, after clicking on password reset link in email
    Search: '/search', // Search routines and users
    Terms: '/terms-and-conditions', // Terms and conditions
    Waitlist: '/join-us' // Form to join waitlist
}