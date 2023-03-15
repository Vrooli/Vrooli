import { Stack, Typography } from "@mui/material";
import { HelpButton } from "components/buttons";
import { noSelect } from "styles";
import { HeaderProps } from "../types";

export const Header = ({
    help,
    sxs,
    title,
}: HeaderProps) => {

    const titleDisplay = <Typography
        component="h1"
        variant="h3"
        sx={{
            textAlign: 'center',
            marginTop: 2,
            marginBottom: 2,
            fontSize: { xs: '1.75rem', sm: '2rem', md: '2.5rem' },
            ...noSelect,
            ...(sxs?.text || {}),
        }}
    >{title}</Typography>

    // If no help data, return a simple title
    if (!help) return titleDisplay;
    // Otherwise, return a stack with the title and help button
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
            {titleDisplay}
            <HelpButton
                markdown={help}
                sx={{ width: '40px', height: '40px' }}
            />
        </Stack>
    )
}