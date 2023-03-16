import { LinearProgress, Stack, Typography } from "@mui/material";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ObjectTitleProps } from "../types";

/**
 * Like PageTitle, but shows a language selector instead of a help button. 
 * Also supports loading state
 */
export const ObjectTitle = ({
    language,
    loading,
    session,
    setLanguage,
    translations,
    title,
    zIndex,
}: ObjectTitleProps) => {

    // Display title or loading bar
    const titleComponent = loading ? <LinearProgress color="inherit" sx={{
        borderRadius: 1,
        width: '50vw',
        height: 8,
        marginTop: '12px !important',
        marginBottom: '12px !important',
        maxWidth: '300px',
    }} /> : <Typography
        component="h1"
        variant="h3"
        sx={{
            textAlign: 'center',
            sx: { marginTop: 2, marginBottom: 2 },
            fontSize: { xs: '1.5rem', sm: '2rem', md: '2.5rem' },
        }}
    >{title}</Typography>

    return (
        <Stack
            direction="row"
            justifyContent="center"
            alignItems="center"
            spacing={2}
            sx={{
                marginTop: 2,
                marginBottom: 2,
                marginLeft: 'auto',
                marginRight: 'auto',
                maxWidth: '700px',
            }}
        >
            {titleComponent}
            <SelectLanguageMenu
                currentLanguage={language}
                handleCurrent={setLanguage}
                session={session}
                translations={translations}
                zIndex={zIndex}
            />
        </Stack>
    )
}