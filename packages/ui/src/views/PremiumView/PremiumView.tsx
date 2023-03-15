import { Button, Stack, Typography } from "@mui/material";
import { TopBar } from "components";
import { useTranslation } from "react-i18next";
import { PremiumViewProps } from "../types";

export const PremiumView = ({
    display = 'page',
    session,
}: PremiumViewProps) => {
    const { t } = useTranslation();

    // TODO convert MaxObjects to list of limit increases 
    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'Premium'
                }}
            />
            <Stack direction="column" spacing={4} justifyContent="center" alignItems="center" sx={{ marginTop: 2 }}>
                {/* Introduction to premium */}
                <Typography variant="h5" sx={{ textAlign: 'center' }}>{t('PremiumIntro1')}</Typography>
                <Typography variant="h5" sx={{ textAlign: 'center' }}>{t('PremiumIntro2')}</Typography>
                {/* Main features as cards */}
                {/* TODO */}
                {/* Link to open popup that displays all limit increases */}
                {/* TODO */}
                {/* Button row for different subscriptions, with donation popup at bottom */}
                <Button
                    fullWidth
                    onClick={() => { }}
                >x/year</Button>
                <Button
                    fullWidth
                    onClick={() => { }}
                >x/month</Button>
                <Button
                    fullWidth
                    onClick={() => { }}
                >One-time donation (no premium)</Button>
            </Stack>
        </>
    )
}