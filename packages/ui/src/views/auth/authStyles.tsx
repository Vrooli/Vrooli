import Box from "@mui/material/Box";
import InputAdornment from "@mui/material/InputAdornment";
import { keyframes } from "@mui/material";
import { styled } from "@mui/material";
import { Button } from "../../components/buttons/Button.js";
import { Divider } from "../../components/layout/Divider.js";
import { OAUTH_PROVIDERS, getOAuthInitRoute } from "@vrooli/shared";
import { IconCommon } from "../../icons/Icons.js";
import { bottomNavHeight } from "../../styles.js";

export const OAUTH_PROVIDERS_INFO = [
    {
        name: "X",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.X),
        site: "https://x.com",
        style: {
            background: "#000000",
            color: "#ffffff",
            hoverBackground: "#14171a",
            border: "none",
        },
    },
    {
        name: "Google",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.Google),
        site: "https://google.com",
        style: {
            background: "#4285f4",
            color: "#ffffff",
            border: "none",
            hoverBackground: "#357ABD",
        },
    },
    {
        name: "Apple",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.Apple),
        site: "https://apple.com",
        style: {
            background: "#000000",
            color: "#ffffff",
            hoverBackground: "#333333",
            border: "none",
        },
    },
    {
        name: "GitHub",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.GitHub),
        site: "https://github.com",
        style: {
            background: "#171a21",
            color: "#ffffff",
            hoverBackground: "#2a475e",
            border: "none",
        },
    },
    {
        name: "Facebook",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.Facebook),
        site: "https://facebook.com",
        style: {
            background: "#1877f2",
            color: "#ffffff",
            hoverBackground: "#0a66c2",
            border: "none",
        },
    },
] as const;

// Type for the OAuth provider style
export interface OAuthProviderStyle {
    background: string;
    color: string;
    hoverBackground: string;
    border: string;
}

// Animation constants
const ANIMATION_DURATION = 0.5;
const SPACING_LARGE = 4;
const SPACING_MEDIUM = 2;

export const fadeIn = keyframes`
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
`;

export const AuthContainer = styled(Box)(({ theme }) => ({
    width: "100%",
    maxWidth: "900px",
    background: theme.palette.background.paper,
    borderRadius: theme.spacing(2),
    boxShadow: "0 8px 32px rgba(0, 0, 0, 0.1)",
    animation: `${fadeIn} ${ANIMATION_DURATION}s ease-out`,
    overflow: "hidden",
    display: "flex",
    flexDirection: "row",
    [theme.breakpoints.down("md")]: {
        maxWidth: "450px",
        flexDirection: "column",
    },
    [theme.breakpoints.down("sm")]: {
        maxWidth: "100%",
    },
}));

export const OuterAuthFormContainer = styled(Box)(({ theme }) => ({
    width: "100%",
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    marginBottom: `calc(${bottomNavHeight} + ${theme.spacing(SPACING_LARGE)})`,
}));

export const AuthFormContainer = styled(Box)(({ theme }) => ({
    flex: "1 1 60%",
    borderRight: `1px solid ${theme.palette.divider}`,
    display: "flex",
    flexDirection: "column",
    margin: "auto",
    [theme.breakpoints.down("md")]: {
        flex: "1 1 auto",
        borderRight: "none",
        borderBottom: `1px solid ${theme.palette.divider}`,
    },
    // Override browser default autofill styles
    "& input:-webkit-autofill, & input:-webkit-autofill:hover, & input:-webkit-autofill:focus, & input:-webkit-autofill:active": {
        transition: "background-color 5000s ease-in-out 0s", // Hack to delay browser background styling
        boxShadow: `0 0 0 30px ${theme.palette.background.paper} inset !important`, // Use box-shadow to mimic background
        WebkitTextFillColor: `${theme.palette.text.primary} !important`, // Ensure text color matches theme
        caretColor: theme.palette.text.primary, // Match cursor color
        borderRadius: "inherit", // Inherit border radius from the input element
    },
}));

export const OAuthContainer = styled(Box)(({ theme }) => ({
    flex: "1 1 40%",
    display: "flex",
    flexDirection: "column",
    justifyContent: "center",
    padding: theme.spacing(SPACING_MEDIUM),
    [theme.breakpoints.down("md")]: {
        flex: "1 1 auto",
        padding: theme.spacing(SPACING_MEDIUM),
    },
}));

// Or divider component using our custom Divider
export const OrDivider = ({ children, ...props }: React.ComponentProps<typeof Divider>) => {
    return (
        <Divider
            {...props}
            className="w-full my-4 md:hidden"
        >
            {typeof children === "string" ? children : "or"}
        </Divider>
    );
};

interface OAuthButtonProps extends React.ComponentProps<typeof Button> {
    providerStyle: OAuthProviderStyle;
}

// OAuth button component using our custom Button
export const OAuthButton = ({ providerStyle, children, ...props }: OAuthButtonProps) => {
    return (
        <Button
            {...props}
            variant="outline"
            size="lg"
            className="w-full rounded-2xl transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg justify-start"
            style={{
                backgroundColor: providerStyle.background,
                color: providerStyle.color,
                border: providerStyle.border,
            }}
            onMouseEnter={(e) => {
                e.currentTarget.style.backgroundColor = providerStyle.hoverBackground;
                props.onMouseEnter?.(e);
            }}
            onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = providerStyle.background;
                props.onMouseLeave?.(e);
            }}
        >
            {children}
        </Button>
    );
};

export const OAuthSection = styled(Box)(({ theme }) => ({
    padding: theme.spacing(0, 2, 2),
    display: "flex",
    flexDirection: "column",
    gap: theme.spacing(2),
}));

export const FormSection = styled(Box)(({ theme }) => ({
    padding: theme.spacing(2),
    width: "100%",
    display: "flex",
    flexDirection: "column",
    gap: theme.spacing(2),
}));

// Common styles as constants
export const baseFormStyle = {
    paddingBottom: "unset",
    margin: "0px",
} as const;

export const breadcrumbsStyle = {
    margin: "auto",
} as const;

export const oAuthSpanStyle = {
    marginRight: "auto",
} as const;

export const contentWrapStyle = {
    minHeight: "100%",
    alignItems: "flex-start",
    flexDirection: "column",
    gap: "16px",
} as const;

export const emailStartAdornment = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="Email" />
        </InputAdornment>
    ),
};

export const nameStartAdornment = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="User" />
        </InputAdornment>
    ),
} as const;
