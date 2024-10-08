import { FocusMode, FocusModeStopCondition, LINKS, MaxObjects } from "@local/shared";
import { SessionContext } from "contexts";
import { FocusModeIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser, getFocusModeInfo } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { SelectorBase } from "../Selector/Selector";

function getFocusModeOptionDescription(focusMode: FocusMode) {
    return focusMode.description;
}

function getFocusModeOptionLabel(focusMode: FocusMode) {
    return focusMode.name;
}

function getDisplayIcon() {
    return <FocusModeIcon />;
}

/**
 * Sets your focus mode. Can also create new focus modes.
 */
export function FocusModeSelector() {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { active, all } = useMemo(() => getFocusModeInfo(session), [session]);

    const { canAdd, hasPremium } = useMemo(() => {
        const { hasPremium } = getCurrentUser(session);
        const max = hasPremium ? MaxObjects.FocusMode.User.premium : MaxObjects.FocusMode.User.noPremium;
        return { canAdd: all.length < max, hasPremium };
    }, [all.length, session]);

    const handleAddNewFocusMode = useCallback(function handleAddNewFocusMode() {
        // If you can add, open settings
        if (canAdd) setLocation(LINKS.SettingsFocusModes);
        // If you can't add and don't have premium, open premium page
        else if (!hasPremium) {
            setLocation(LINKS.Pro);
            PubSub.get().publish("snack", { message: "Upgrade to increase limit", severity: "Info" });
        }
        // Otherwise, show error
        else PubSub.get().publish("snack", { message: "Max reached", severity: "Error" });
    }, [canAdd, hasPremium, setLocation]);

    const handleChangedFocusMode = useCallback(function handleChangedFocusMode(newMode: FocusMode) {
        newMode && PubSub.get().publish("focusMode", {
            __typename: "ActiveFocusMode" as const,
            mode: newMode,
            stopCondition: FocusModeStopCondition.NextBegins,
        });
    }, []);

    const addOption = useMemo(function addOptionMemo() {
        return {
            label: t("AddFocusMode"),
            onSelect: handleAddNewFocusMode,
        };
    }, [handleAddNewFocusMode, t]);

    return (
        <SelectorBase
            name="active"
            options={all}
            getDisplayIcon={getDisplayIcon}
            getOptionLabel={getFocusModeOptionLabel}
            getOptionDescription={getFocusModeOptionDescription}
            fullWidth={true}
            inputAriaLabel="Focus Mode"
            label={t("FocusMode", { count: 1, defaultValue: "Focus Mode" })}
            onChange={handleChangedFocusMode}
            addOption={addOption}
            value={active?.mode ?? null}
        />
    );
}
