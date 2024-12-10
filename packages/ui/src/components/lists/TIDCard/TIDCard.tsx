import { Box, BoxProps, Button, Typography, styled, useTheme } from "@mui/material";
import { WarningIcon } from "icons";
import { TIDCardProps, TIDCardSize } from "../types";

interface OuterCardProps extends BoxProps {
    isClickable: boolean;
    size: TIDCardSize | undefined;
}

const OuterCard = styled(Box, {
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

interface IconBoxProps {
    size: TIDCardSize | undefined;
}

const IconBox = styled(Box, {
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

interface TextBoxProps {
    size: TIDCardSize | undefined;
}

const TextBox = styled(Box, {
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

const WarningBox = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    color: theme.palette.warning.main,
    marginTop: 1,
}));

const SelectButton = styled(Button)(({ theme }) => ({
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

/**
 * A card with a title, description, and icon
 */
export function TIDCard({
    below,
    buttonText,
    description,
    Icon,
    id,
    onClick,
    size,
    title,
    warning,
    ...props
}: TIDCardProps) {
    const { palette } = useTheme();
    const isSmall = size === "small";

    return (
        <OuterCard
            {...props}
            id={id}
            isClickable={typeof onClick === "function"}
            onClick={onClick}
            size={size}
        >
            {Icon && <IconBox size={size}>
                <Icon width={"50px"} height={"50px"} fill={palette.background.textPrimary} />
            </IconBox>}
            <TextBox size={size}>
                <Box>
                    <Typography variant='h6' component='div' sx={titleTextStyle}>
                        {title}
                    </Typography>
                    <Typography variant='body2' color={palette.background.textSecondary} sx={descriptionTextStyle}>
                        {description}
                    </Typography>
                    {warning && (
                        <WarningBox>
                            <WarningIcon style={warningIconStyle} />
                            <Typography variant="body2">{warning}</Typography>
                        </WarningBox>
                    )}
                </Box>
                {below}
                {buttonText && (
                    <SelectButton
                        size={isSmall ? "small" : "medium"}
                        variant="text"
                    >
                        {buttonText}
                    </SelectButton>
                )}
            </TextBox>
        </OuterCard>
    );
}
