import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { ScrollBox } from "../../styles.js";
import { SiteStatisticsPanel } from "../admin/components/SiteStatisticsPanel.js";
import { type StatsSiteViewProps } from "../types.js";

/**
 * Displays site-wide statistics using the enhanced SiteStatisticsPanel
 */
export function StatsSiteView({
    display,
}: StatsSiteViewProps) {
    const { t } = useTranslation();

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <Navbar title={t("StatisticsShort")} />
                <SiteStatisticsPanel />
            </ScrollBox>
        </PageContainer>
    );
}
