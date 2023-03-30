import { Button, Paper, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, useTheme } from "@mui/material";
import { openLink, useLocation } from "@shared/route";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useTranslation } from "react-i18next";
import { PremiumViewProps } from "../types";

function createData(feature, nonPremium, premium) {
    return { feature, nonPremium, premium };
}

const rows = [
    createData('Routines and processes', 'Up to 25 private, 100 public', 'Very high limits'),
    createData('AI-related features', 'GPT API key required', '✔️'),
    createData('Human and bot collaboration', 'GPT API key required', '✔️'),
    createData('Customize and replicate public organizations', '✔️', '✔️'),
    createData('Copy and adapt public routines', '✔️', '✔️'),
    createData('Analytics dashboard', 'Essential', 'Advanced'),
    createData('Customizable user experience', '✔️', '✔️'),
    createData('Community sharing', '✔️', '✔️'),
    createData('Data import/export', '✔️', '✔️'),
    createData('Industry-standard templates', '✔️', '✔️'),
    createData('Mobile app', '✔️', '✔️'),
    createData('Tutorial resources', '✔️', '✔️'),
    createData('Community support', '✔️', '✔️'),
    createData('Updates and improvements', '✔️', 'Early access'),
    createData('Task management', '✔️', '✔️'),
    createData('Calendar integration', '✔️', '✔️'),
    createData('Customizable notifications', '✔️', '✔️'),
    createData('Provide feedback', '✔️', '✔️'),
    createData('Ad-free experience', '❌', '✔️'),
    createData('Enhanced focus modes', '❌', '✔️'),
    createData('Premium API access', '❌', '✔️'),
];

const yearlyPaymentsLink = 'https://buy.stripe.com/test_eVafZvgiFg2Zgx2000';
const monthlyPaymentsLink = 'https://buy.stripe.com/test_eVafZvgiFg2Zgx2000';
const donationLink = 'https://buy.stripe.com/test_eVafZvgiFg2Zgx2000';

export const PremiumView = ({
    display = 'page',
}: PremiumViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

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
                <Typography variant="h6" sx={{ textAlign: 'center' }}>{t('PremiumIntro1')}</Typography>
                <Typography variant="h6" sx={{ textAlign: 'center' }}>{t('PremiumIntro2')}</Typography>
                {/* Main features as table */}
                <TableContainer component={Paper} sx={{ maxWidth: 800 }}>
                    <Table aria-label="features table">
                        <TableHead sx={{ background: palette.primary.light }}>
                            <TableRow>
                                <TableCell sx={{ color: palette.primary.contrastText }}>Feature</TableCell>
                                <TableCell align="center" sx={{ color: palette.primary.contrastText }}>Non-Premium</TableCell>
                                <TableCell align="center" sx={{ color: palette.primary.contrastText }}>Premium</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {rows.map((row) => (
                                <TableRow key={row.feature}>
                                    <TableCell component="th" scope="row">
                                        {row.feature}
                                    </TableCell>
                                    <TableCell align="center">{row.nonPremium}</TableCell>
                                    <TableCell align="center">{row.premium}</TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </TableContainer>
                <Typography variant="body1" sx={{ textAlign: 'center' }}>
                    By upgrading to a Vrooli Premium account, you'll not only benefit from an ad-free experience and increased limits but also gain access to AI-powered integrations, advanced analytics tools, enhanced focus modes, premium API access, and exclusive early access to new features. These premium features are designed to elevate your efficiency, productivity, and overall success. Don't miss out on the opportunity to maximize the potential of your Vrooli experience – go Premium today!
                </Typography>
                {/* Link to open popup that displays all limit increases */}
                {/* TODO */}
                {/* Button row for different subscriptions, with donation option at bottom. Should have dialog like on matthalloran.info */}
                <Button
                    fullWidth
                    href={yearlyPaymentsLink}
                    onClick={() => openLink(setLocation, yearlyPaymentsLink)}
                >$149.99/year</Button>
                <Button
                    fullWidth
                    href={monthlyPaymentsLink}
                    onClick={() => openLink(setLocation, monthlyPaymentsLink)}
                >$14.99/month</Button>
                <Button
                    fullWidth
                    href={donationLink}
                    onClick={() => openLink(setLocation, donationLink)}
                >One-time donation (no premium)</Button>
            </Stack>
        </>
    )
}