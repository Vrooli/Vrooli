import { 
    Assessment, 
    People, 
    Settings, 
    Report, 
    IntegrationInstructions,
    Dashboard as DashboardIcon 
} from "@mui/icons-material";
import { 
    Box, 
    Container, 
    Tab, 
    Tabs, 
    Typography, 
    Paper,
    Alert,
    Chip,
    Stack
} from "@mui/material";
import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { AdminRoute } from "../../components/auth/AdminRoute.js";
import { PageContainer } from "../../components/Page/Page.js";
import { useIsAdmin } from "../../hooks/useIsAdmin.js";
import { SiteStatisticsPanel } from "./components/SiteStatisticsPanel.js";
import { UserManagementPanel } from "./components/UserManagementPanel.js";
import { ExternalServicesPanel } from "./components/ExternalServicesPanel.js";
import { ReportsPanel } from "./components/ReportsPanel.js";
import { SystemSettingsPanel } from "./components/SystemSettingsPanel.js";

interface TabPanelProps {
    children?: React.ReactNode;
    index: number;
    value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index }) => (
    <div
        role="tabpanel"
        hidden={value !== index}
        id={`admin-tabpanel-${index}`}
        aria-labelledby={`admin-tab-${index}`}
    >
        {value === index && <Box sx={{ py: 3 }}>{children}</Box>}
    </div>
);

const a11yProps = (index: number) => ({
    id: `admin-tab-${index}`,
    "aria-controls": `admin-tabpanel-${index}`,
});

/**
 * Main admin dashboard component with tabbed interface for different admin functions
 */
export const AdminView: React.FC = () => {
    const { t } = useTranslation();
    const { isAdmin, adminUser } = useIsAdmin();
    const [activeTab, setActiveTab] = useState(0);

    const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
        setActiveTab(newValue);
    };

    if (!isAdmin) {
        return (
            <PageContainer>
                <Alert severity="error" sx={{ m: 2 }}>
                    <Typography variant="h6">Access Denied</Typography>
                    <Typography>
                        Administrator privileges are required to access this page.
                    </Typography>
                </Alert>
            </PageContainer>
        );
    }

    return (
        <PageContainer>
            <Container maxWidth="xl" sx={{ py: 2 }}>
                {/* Header */}
                <Box sx={{ mb: 4 }}>
                    <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 2 }}>
                        <DashboardIcon color="primary" sx={{ fontSize: 32 }} />
                        <Typography variant="h3" component="h1">
                            {t("AdminDashboard")}
                        </Typography>
                        <Chip 
                            label={t("Administrator")} 
                            color="primary" 
                            variant="outlined"
                        />
                    </Stack>
                    
                    <Typography variant="subtitle1" color="text.secondary">
                        {t("AdminDashboardDescription")}
                    </Typography>
                    
                    {adminUser && (
                        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                            {t("LoggedInAs")}: {adminUser.name || adminUser.id}
                        </Typography>
                    )}
                </Box>

                {/* Navigation Tabs */}
                <Paper sx={{ mb: 3 }}>
                    <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
                        <Tabs 
                            value={activeTab} 
                            onChange={handleTabChange} 
                            aria-label="admin dashboard tabs"
                            variant="scrollable"
                            scrollButtons="auto"
                        >
                            <Tab
                                icon={<Assessment />}
                                label={t("SiteStatistics")}
                                {...a11yProps(0)}
                            />
                            <Tab
                                icon={<People />}
                                label={t("UserManagement")}
                                {...a11yProps(1)}
                            />
                            <Tab
                                icon={<IntegrationInstructions />}
                                label={t("ExternalServices")}
                                {...a11yProps(2)}
                            />
                            <Tab
                                icon={<Report />}
                                label={t("ReportsModeration")}
                                {...a11yProps(3)}
                            />
                            <Tab
                                icon={<Settings />}
                                label={t("SystemSettings")}
                                {...a11yProps(4)}
                            />
                        </Tabs>
                    </Box>

                    {/* Tab Panels */}
                    <TabPanel value={activeTab} index={0}>
                        <SiteStatisticsPanel />
                    </TabPanel>

                    <TabPanel value={activeTab} index={1}>
                        <UserManagementPanel />
                    </TabPanel>

                    <TabPanel value={activeTab} index={2}>
                        <ExternalServicesPanel />
                    </TabPanel>

                    <TabPanel value={activeTab} index={3}>
                        <ReportsPanel />
                    </TabPanel>

                    <TabPanel value={activeTab} index={4}>
                        <SystemSettingsPanel />
                    </TabPanel>
                </Paper>
            </Container>
        </PageContainer>
    );
};

/**
 * Admin view wrapped with route protection
 */
export const ProtectedAdminView: React.FC = () => (
    <AdminRoute>
        <AdminView />
    </AdminRoute>
);