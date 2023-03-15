import { Stack, Typography, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons";
import { noSelect } from "styles";
import { SubheaderProps } from "../types";

export const Subheader = ({
    help,
    Icon,
    sxs,
    title,
}: SubheaderProps) => {
    const { palette } = useTheme();

    const titleDisplay = <Typography
        component="h2"
        variant="h4"
        sx={{
            textAlign: 'center',
            marginTop: 2,
            marginBottom: 2,
            fontSize: { xs: '1.5rem', sm: '1.75rem', md: '2rem' },
            ...noSelect,
            ...(sxs?.text || {}),
        }}
    >{title}</Typography>

    // If no help or Icon data, return a simple title
    if (!help && !Icon) return titleDisplay;
    // Otherwise, return a stack with Icon, title, and help button
    return (
        <Stack
            direction="row"
            justifyContent="center"
            alignItems="center"
            sx={{
                marginTop: 2,
                marginBottom: 2,
                ...noSelect,
                ...(sxs?.stack || {}),
            }}
        >
            {Icon && <Icon fill={palette.background.textPrimary} style={{ width: '30px', height: '30px', marginRight: 8 }} />}
            {titleDisplay}
            {help && <HelpButton
                markdown={help}
                sx={{ width: '30px', height: '30px' }}
            />}
        </Stack>
    )
}