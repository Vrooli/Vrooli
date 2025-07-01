import { Box, Button, Container, Grid, Stack, Tab, Tabs } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { ScrollBox } from "../../styles.js";
import { SiteStatisticsPanel } from "../admin/components/SiteStatisticsPanel.js";
import { SystemHealthCard } from "../../components/stats/SystemHealthCard.js";
import { SystemMetricsCards } from "../../components/stats/SystemMetricsCards.js";
import { useSystemHealth } from "../../hooks/useSystemHealth.js";
import { IconCommon } from "../../icons/Icons.js";
import { type StatsSiteViewProps } from "../types.js";

/**
 * Displays site-wide statistics with tabs for different types of data
 */
export function StatsSiteView({
    display,
}: StatsSiteViewProps) {
    const { t } = useTranslation();
    const [currentTab, setCurrentTab] = useState(0);
    const { 
        healthData, 
        metricsData, 
        healthLoading, 
        metricsLoading, 
        healthError, 
        metricsError, 
        refresh, 
    } = useSystemHealth();

    const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
        setCurrentTab(newValue);
    };

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <Navbar title={t("StatisticsShort")} />
                <Container maxWidth="xl" sx={{ py: 2 }}>
                    <Box sx={{ borderBottom: 1, borderColor: "divider", mb: 3 }}>
                        <Tabs value={currentTab} onChange={handleTabChange}>
                            <Tab label="Site Statistics" />
                            <Tab label="System Health" />
                        </Tabs>
                    </Box>

                    {currentTab === 0 && (
                        <SiteStatisticsPanel />
                    )}

                    {currentTab === 1 && (
                        <Stack spacing={3}>
                            <Box sx={{ display: "flex", justifyContent: "flex-end" }}>
                                <Button
                                    variant="outlined"
                                    onClick={refresh}
                                    disabled={healthLoading || metricsLoading}
                                    startIcon={<IconCommon name="Refresh" size={16} />}
                                >
                                    Refresh
                                </Button>
                            </Box>
                            
                            <SystemMetricsCards 
                                metricsData={metricsData}
                                loading={metricsLoading}
                                error={metricsError}
                            />
                            
                            <SystemHealthCard
                                healthData={healthData}
                                loading={healthLoading}
                                error={healthError}
                                showDetails={true}
                            />
                        </Stack>
                    )}
                </Container>
            </ScrollBox>
        </PageContainer>
    );
}
