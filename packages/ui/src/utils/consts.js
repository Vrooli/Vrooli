import { COOKIE, ORDER_STATUS } from '@local/shared';

export const ORDER_FILTERS = [
    {
        label: 'All',
        value: 'All',
    },
    {
        label: 'Canceled by Admin',
        value: ORDER_STATUS.CanceledByAdmin,
    },
    {
        label: 'Canceled by Customer',
        value: ORDER_STATUS.CanceledByCustomer,
    },
    {
        label: 'Pending Cancel',
        value: ORDER_STATUS.PendingCancel,
    },
    {
        label: 'Rejected',
        value: ORDER_STATUS.Rejected,
    },
    {
        label: 'Pending',
        value: ORDER_STATUS.Pending,
    },
    {
        label: 'Approved',
        value: ORDER_STATUS.Approved,
    },
    {
        label: 'Scheduled',
        value: ORDER_STATUS.Scheduled,
    },
    {
        label: 'In Transit',
        value: ORDER_STATUS.InTransit,
    },
    {
        label: 'Delivered',
        value: ORDER_STATUS.Delivered,
    },
]

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
    AdminGallery: "/admin/gallery",
    AdminHero: "/admin/hero",
    AdminInventory: "/admin/inventory",
    AdminOrders: "/admin/orders",
    Cart: "/cart",
    ForgotPassword: "/forgot-password",
    Gallery: "/gallery",
    Home: "/",
    LogIn: "/login",
    PrivacyPolicy: "/privacy-policy",
    Profile: "/profile",
    Register: "/register",
    ResetPassword: "/password-reset",
    Shopping: "/shopping",
    Terms: "/terms-and-conditions",
}