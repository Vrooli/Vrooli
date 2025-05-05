import { endpointsRun, Run } from "@local/shared";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { useObjectActions } from "../../../hooks/objectActions.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useLocation } from "../../../route/router.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { RunViewProps } from "./types.js";

export function RunView({
    display,
    onClose,
}: RunViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { object: existing, isLoading, setObject: setRun } = useManagedObject<Run>({
        ...endpointsRun.findOne,
        objectType: "Run",
    });

    const { title } = useMemo(() => ({ title: getDisplay(existing).title ?? "" }), [existing]);

    const actionData = useObjectActions({
        object: existing,
        objectType: "Run",
        setLocation,
        setObject: setRun,
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
