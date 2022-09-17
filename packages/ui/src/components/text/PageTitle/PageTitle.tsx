import { Stack, Typography } from "@mui/material";
import { HelpButton } from "components/buttons";
import { PageTitleProps } from "../types";

export const PageTitle = ({
    helpText,
    title,
    sxs,
}: PageTitleProps) => {
    if (!helpText) {
        return (
            <Typography
                component="h1"
                variant="h3"
                sx={{
                    textAlign: 'center',
                    sx: { marginTop: 2, marginBottom: 2 },
                    fontSize: { xs: '2rem', sm: '2.5rem', md: '3rem' },
                    ...(sxs?.text || {}),
                }}
            >{title}</Typography>
        )
    }
    return (
        <Stack
            direction="row"
            justifyContent="center"
            alignItems="center"
            sx={{
                marginTop: 2,
                marginBottom: 2,
                ...(sxs?.stack || {}),
            }}
        >
            <Typography
                component="h1"
                variant="h3"
                sx={{
                    textAlign: 'center',
                    fontSize: { xs: '2rem', sm: '2.5rem', md: '3rem' },
                    ...(sxs?.text || {}),
                }}
            >{title}</Typography>
            <HelpButton markdown={helpText} sx={{ width: '40px', height: '40px' }} />
        </Stack>
    )
}