/**
 * Handles selecting a run from a list of runs.
 */

import { useCallback, useMemo } from "react";
import { ListMenuItemData, SelectRoutineTypeMenuProps } from "../types";
import { APP_LINKS } from "@shared/consts";
import { useLocation } from "@shared/route";
import { ListMenu } from "../ListMenu/ListMenu";
import { PubSub, stringifySearchParams } from "utils";
import { SnackSeverity } from "components";
import { getCurrentUser } from "utils/authentication";

const options: ListMenuItemData<string>[] = [
    { label: 'Routine (Single Step)', value: `${APP_LINKS.Routine}/add` },
    { label: 'Routine (Multi Step)', value: `${APP_LINKS.Routine}/add?build=true` },
]

export const SelectRoutineTypeMenu = ({
    anchorEl,
    handleClose,
    session,
    zIndex,
}: SelectRoutineTypeMenuProps) => {
    const [, setLocation] = useLocation();

    const handleSelect = useCallback((path: string) => {
        setLocation(path);
    }, [setLocation]);

    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    if (Boolean(anchorEl) && !userId) {
        PubSub.get().publishSnack({ message: 'Must be logged in.', severity: SnackSeverity.Error });
        setLocation(`${APP_LINKS.Start}${stringifySearchParams({
            redirect: window.location.pathname
        })}`);
    }

    return (
        <ListMenu
            id={`select-routine-type-menu`}
            anchorEl={anchorEl}
            title='Routine Type'
            data={options}
            onSelect={handleSelect}
            onClose={handleClose}
            zIndex={zIndex}
        />
    );
}