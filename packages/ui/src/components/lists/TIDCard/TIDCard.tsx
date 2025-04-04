import { Box, BoxProps, Button, Typography, styled, useTheme } from "@mui/material";
import { Icon, IconCommon } from "../../../icons/Icons.js";
import { TIDCardProps, TIDCardSize } from "../types.js";

interface OuterCardProps extends BoxProps {
    isClickable: boolean;
    size?: TIDCardSize | undefined;
}

export const TIDCardBase = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isClickable" && prop !== "size",
})<OuterCardProps>(({ isClickable, size, theme }) => {
    const isSmall = size === "small";

    return {
        width: "100%",
        padding: isSmall ? theme.spacing(0.5) : theme.spacing(1),
        cursor: isClickable ? "pointer" : "default",
        background: theme.palette.background.paper,
        "&:hover": {
            filter: "brightness(1.05)",
        },
        display: "flex",
        boxShadow: theme.shadows[2],
        borderRadius: theme.spacing(1),
        [theme.breakpoints.down("sm")]: {
            borderBottom: `1px solid ${theme.palette.divider}`,
            boxShadow: "none",
            borderRadius: 0,
        },
    } as const;
});

interface IconBoxProps extends BoxProps {
    size?: TIDCardSize;
}

export const TIDIconBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "size",
})<IconBoxProps>(({ size, theme }) => {
    const isSmall = size === "small";

    return {
        width: isSmall ? "50px" : "75px",
        height: "100%",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: isSmall ? theme.spacing(1) : theme.spacing(2),
        [theme.breakpoints.down("sm")]: {
            padding: isSmall ? theme.spacing(0.5) : theme.spacing(1),
            paddingRight: isSmall ? theme.spacing(1) : theme.spacing(2),
        },
    } as const;
});

export interface TIDIconProps {
    iconInfo: NonNullable<TIDCardProps["iconInfo"]>;
    size?: TIDCardSize;
}

export function TIDIcon({ iconInfo, size }: TIDIconProps) {
    const { palette } = useTheme();

    return (
        <TIDIconBox size={size}>
            <Icon
                decorative
                fill={palette.background.textPrimary}
                info={iconInfo}
                size={50}
            />
        </TIDIconBox>
    );
}

interface TextBoxProps extends BoxProps {
    size?: TIDCardSize;
}

export const TIDTextBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "size",
})<TextBoxProps>(({ size, theme }) => {
    const isSmall = size === "small";

    return {
        flexGrow: 1,
        height: "100%",
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
        padding: isSmall ? theme.spacing(1) : theme.spacing(2),
        [theme.breakpoints.down("sm")]: {
            padding: isSmall ? theme.spacing(0.5) : theme.spacing(1),
        },
    } as const;
});

export const TIDWarningBox = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    color: theme.palette.warning.main,
    marginTop: 1,
}));

export const TIDSelectButton = styled(Button)(({ theme }) => ({
    marginLeft: "auto",
    alignSelf: "flex-end",
    marginTop: theme.spacing(2),
    [theme.breakpoints.down("sm")]: {
        marginTop: theme.spacing(1),
    },
}));

const titleTextStyle = { overflowWrap: "anywhere" } as const;
const descriptionTextStyle = { overflowWrap: "anywhere" } as const;
const warningIconStyle = { fontSize: 20, marginRight: "8px" } as const;

export interface TIDContentProps {
    title: string;
    description?: string;
    warning?: string;
}

export function TIDContent({ title, description, warning }: TIDContentProps) {
    const { palette } = useTheme();

    return (
        <Box>
            <Typography variant='h6' component='div' sx={titleTextStyle}>
                {title}
            </Typography>
            {description && (
                <Typography variant='body2' color={palette.background.textSecondary} sx={descriptionTextStyle}>
                    {description}
                </Typography>
            )}
            {warning && (
                <TIDWarningBox>
                    <IconCommon
                        name="Warning"
                        style={warningIconStyle}
                    />
                    <Typography variant="body2">{warning}</Typography>
                </TIDWarningBox>
            )}
        </Box>
    );
}

export interface TIDButtonProps {
    buttonText: string;
    size?: TIDCardSize;
    onClick?: React.MouseEventHandler<HTMLButtonElement>;
}

export function TIDButton({ buttonText, size, onClick }: TIDButtonProps) {
    const isSmall = size === "small";

    return (
        <TIDSelectButton
            onClick={onClick}
            size={isSmall ? "small" : "medium"}
            variant="text"
        >
            {buttonText}
        </TIDSelectButton>
    );
}

/**
 * A card with a title, description, and icon
 */
export function TIDCard({
    below,
    buttonText,
    description,
    iconInfo,
    id,
    onClick,
    size,
    title,
    warning,
    ...props
}: TIDCardProps) {
    return (
        <TIDCardBase
            {...props}
            id={id}
            isClickable={typeof onClick === "function"}
            onClick={onClick}
            size={size}
        >
            {iconInfo && <TIDIcon iconInfo={iconInfo} size={size} />}
            <TIDTextBox size={size}>
                <TIDContent
                    title={title}
                    description={description}
                    warning={warning}
                />
                {below}
                {buttonText && (
                    <TIDButton
                        buttonText={buttonText}
                        size={size}
                    />
                )}
            </TIDTextBox>
        </TIDCardBase>
    );
}
