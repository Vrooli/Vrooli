import { LINKS } from "@local/shared";
import { useTranslation } from "react-i18next";
import { BreadcrumbsBase } from "../BreadcrumbsBase/BreadcrumbsBase";
import { PolicyBreadcrumbsProps } from "../types";

export const PolicyBreadcrumbs = ({
    ...props
}: PolicyBreadcrumbsProps) => {
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
