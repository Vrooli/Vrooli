import { LINKS } from "@local/consts";
import { useTranslation } from "react-i18next";
import { BreadcrumbsBase } from "../BreadcrumbsBase/BreadcrumbsBase";
export const PolicyBreadcrumbs = ({ ...props }) => {
    const { t } = useTranslation();
    return BreadcrumbsBase({
        paths: [
            [t("Privacy"), LINKS.Privacy],
            [t("Terms"), LINKS.Terms],
        ].map(row => ({ text: row[0], link: row[1] })),
        ariaLabel: "Policies breadcrumb",
        ...props,
    });
};
//# sourceMappingURL=PolicyBreadcrumbs.js.map