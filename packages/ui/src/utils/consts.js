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
    About: "/about",
    Admin: "/admin",
    AdminContactInfo: "/admin/contact-info",
    AdminCustomers: "/admin/customers",
    AdminHero: "/admin/hero",
    ForgotPassword: "/forgot-password",
    Home: "/",
    LogIn: "/login",
    PrivacyPolicy: "/privacy-policy",
    Profile: "/profile",
    Register: "/register",
    ResetPassword: "/password-reset",
    Terms: "/terms-and-conditions",
    Waitlist: "/join-us"
}