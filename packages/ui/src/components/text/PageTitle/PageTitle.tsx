import { Stack, Typography } from "@mui/material";
import { HelpButton } from "components/buttons";
import { noSelect } from "styles";
import { PageTitleProps } from "../types";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils";

export const PageTitle = ({
    helpKey,
    helpVariables,
    titleKey,
    titleVariables,
    session,
    sxs,
}: PageTitleProps) => {
    const { t } = useTranslation();

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
            >{t(`common:${titleKey}`, { lng: getUserLanguages(session)[0], ...(titleVariables ?? {}) })}</Typography>
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
            >{t(`common:${titleKey}`, titleVariables)}</Typography>
            <HelpButton
                markdown={t(`common:${helpKey}`, { lng: getUserLanguages(session)[0], ...(helpVariables ?? {}) })}
                sx={{ width: '40px', height: '40px' }}
            />
        </Stack>
    )
}