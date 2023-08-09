import { endpointGetRunProject, RunProject } from "@local/shared";
import { useTheme } from "@mui/material";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getDisplay } from "utils/display/listTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { RunProjectViewProps } from "../types";

export const RunProjectView = ({
    display = "page",
    onClose,
    zIndex,
}: RunProjectViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { object: existing, isLoading, setObject: setRunProject } = useObjectFromUrl<RunProject>({
        ...endpointGetRunProject,
        objectType: "RunProject",
    });

    const { title } = useMemo(() => ({ title: getDisplay(existing).title ?? "" }), [existing]);

    useEffect(() => {
        document.title = `${title} | Vrooli`;
    }, [title]);

    const actionData = useObjectActions({
        object: existing,
        objectType: "RunProject",
        setLocation,
        setObject: setRunProject,
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Run", { count: 1 })}
                zIndex={zIndex}
            />
            <>
                {/* TODO */}
            </>
        </>
    );
};
