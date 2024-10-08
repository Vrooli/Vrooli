import { Box, BoxProps, Button, Typography, styled, useTheme } from "@mui/material";
import { WarningIcon } from "icons";
import { TIDCardProps } from "../types";

interface OuterCardProps extends BoxProps {
    isClickable: boolean;
}

const OuterCard = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isClickable",
})<OuterCardProps>(({ isClickable, theme }) => ({
    width: "100%",
    padding: 1,
    cursor: isClickable ? "pointer" : "default",
    background: theme.palette.background.paper,
    "&:hover": {
        filter: "brightness(1.05)",
    },
    display: "flex",
    boxShadow: theme.shadows[4],
    borderRadius: theme.spacing(1),
    [theme.breakpoints.down("sm")]: {
        borderBottom: `1px solid ${theme.palette.divider}`,
        boxShadow: "none",
        borderRadius: 0,
    },
}));

const IconBox = styled(Box)(({ theme }) => ({
    width: "75px",
    height: "100%",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    padding: theme.spacing(2),
    [theme.breakpoints.down("sm")]: {
        padding: theme.spacing(1),
        paddingRight: theme.spacing(2),
    },
}));

const TextBox = styled(Box)(({ theme }) => ({
    flexGrow: 1,
    height: "100%",
    display: "flex",
    flexDirection: "column",
    justifyContent: "space-between",
    padding: theme.spacing(2),
    [theme.breakpoints.down("sm")]: {
        padding: theme.spacing(1),
    },
}));

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
    title,
    warning,
    ...props
}: TIDCardProps) {
    const { palette } = useTheme();

    return (
        <OuterCard
            {...props}
            id={id}
            isClickable={typeof onClick === "function"}
            onClick={onClick}
        >
            {Icon && <IconBox>
                <Icon width={"50px"} height={"50px"} fill={palette.background.textPrimary} />
            </IconBox>}
            <TextBox>
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
                {buttonText && <SelectButton
                    size='small'
                    variant="text"
                >{buttonText}</SelectButton>}
            </TextBox>
        </OuterCard>
    );
}
