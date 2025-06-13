import { IconCommon } from "../../icons/Icons.js";
import { 
    Box, 
    Container, 
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
import { CreditStatsPanel } from "./CreditStatsPanel.js";
import { ErrorBoundary } from "../../components/ErrorBoundary/ErrorBoundary.js";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { Navbar } from "../../components/navigation/Navbar.js";


interface AdminViewProps {
    display: "Page" | "Dialog" | "Partial";
}

/**
 * Main admin dashboard component with tabbed interface for different admin functions
 */
export const AdminView: React.FC<AdminViewProps> = ({ display = "Page" }) => {
    const { t } = useTranslation();
    const { isAdmin, adminUser } = useIsAdmin();
    
    const tabOptions = [
        { key: "statistics", label: t("SiteStats"), index: 0, iconInfo: { name: "Stats" as const, type: "Common" as const } },
        { key: "users", label: t("User", { count: 2 }), index: 1, iconInfo: { name: "Team" as const, type: "Common" as const } },
        { key: "services", label: t("ExternalServices"), index: 2, iconInfo: { name: "External" as const, type: "Common" as const } },
        { key: "reports", label: t("Report", { count: 2 }), index: 3, iconInfo: { name: "Report" as const, type: "Common" as const } },
        { key: "settings", label: t("Settings"), index: 4, iconInfo: { name: "Settings" as const, type: "Common" as const } },
        { key: "credits", label: t("CreditsAndDonations"), index: 5, iconInfo: { name: "AccountBalance" as const, type: "Common" as const } },
    ];
    const [currTab, setCurrTab] = useState(tabOptions[0]);

    const handleTabChange = (event: React.SyntheticEvent, tab: typeof tabOptions[0]) => {
        setCurrTab(tab);
    };

    if (!isAdmin) {
        return (
            <PageContainer>
                <Alert severity="error" sx={{ m: 2 }}>
                    <Typography variant="h6">{t("AccessDenied")}</Typography>
                    <Typography>
                        {t("AdminPrivilegesRequired")}
                    </Typography>
                </Alert>
            </PageContainer>
        );
    }

    return (
        <PageContainer>
            <Navbar keepVisible title={t("Dashboard")} />
            <Container maxWidth="xl" sx={{ py: 2 }}>
                {/* Admin Status */}
                <Box sx={{ mb: 4 }}>
                    <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 2 }}>
                        <Chip 
                            label={t("Admin")} 
                            color="primary" 
                            variant="outlined"
                            icon={<IconCommon name="Stats" size={18} />}
                        />
                    </Stack>
                    
                    <Typography variant="subtitle1" color="text.secondary">
                        {t("AdminDashboardDescription")}
                    </Typography>
                    
                    {adminUser && (
                        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                            {t("LoggedInAs", { name: adminUser.name || adminUser.id })}
                        </Typography>
                    )}
                </Box>

                {/* Navigation Tabs */}
                <Paper sx={{ mb: 3 }}>
                    <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
                        <PageTabs
                            ariaLabel="admin-dashboard-tabs"
                            currTab={currTab}
                            onChange={handleTabChange}
                            tabs={tabOptions}
                            fullWidth={false}
                        />
                    </Box>

                    {/* Tab Content */}
                    <Box sx={{ py: 3 }}>
                        <ErrorBoundary>
                            {currTab.index === 0 && <SiteStatisticsPanel />}
                            {currTab.index === 1 && <UserManagementPanel />}
                            {currTab.index === 2 && <ExternalServicesPanel />}
                            {currTab.index === 3 && <ReportsPanel />}
                            {currTab.index === 4 && <SystemSettingsPanel />}
                            {currTab.index === 5 && <CreditStatsPanel />}
                        </ErrorBoundary>
                    </Box>
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