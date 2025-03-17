import { endpointsRunProject, RunProject } from "@local/shared";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { useObjectActions } from "hooks/objectActions.js";
import { useManagedObject } from "hooks/useManagedObject.js";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route/router.js";
import { getDisplay } from "utils/display/listTools.js";
import { firstString } from "utils/display/stringTools.js";
import { RunProjectViewProps } from "./types.js";

export function RunProjectView({
    display,
    onClose,
}: RunProjectViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { object: existing, isLoading, setObject: setRunProject } = useManagedObject<RunProject>({
        ...endpointsRunProject.findOne,
        objectType: "RunProject",
    });

    const { title } = useMemo(() => ({ title: getDisplay(existing).title ?? "" }), [existing]);

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
                title={firstString(title, t("Run", { count: 1 }))}
            />
            <>
                {/* TODO */}
            </>
        </>
    );
}
