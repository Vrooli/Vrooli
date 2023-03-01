import { Stack, Typography } from "@mui/material";
import { HelpButton } from "components/buttons";
import { noSelect } from "styles";
import { PageTitleProps } from "../types";
import { useTranslation } from "react-i18next";

export const PageTitle = ({
    helpKey,
    helpVariables,
    titleKey,
    titleVariables,
    sxs,
}: PageTitleProps) => {
    const { t } = useTranslation();

    // If no help data, return a simple title
    if (!helpKey) {
        return (
            <Typography
                component="h1"
                variant="h3"
                sx={{
                    textAlign: 'center',
                    sx: { marginTop: 2, marginBottom: 2 },
                    fontSize: { xs: '2rem', sm: '2.5rem', md: '3rem' },
                    ...noSelect,
                    ...(sxs?.text || {}),
                }}
            >{t(titleKey, {...titleVariables, defaultValue: titleKey })}</Typography>
        )
    }
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
            <Typography
                component="h1"
                variant="h3"
                sx={{
                    textAlign: 'center',
                    fontSize: { xs: '2rem', sm: '2.5rem', md: '3rem' },
                    ...(sxs?.text || {}),
                }}
            >{t(titleKey, {...titleVariables, defaultValue: titleKey })}</Typography>
            <HelpButton
                markdown={t(helpKey, {...helpVariables, defaultValue: helpKey })}
                sx={{ width: '40px', height: '40px' }}
            />
        </Stack>
    )
}