import { FocusMode, FocusModeStopCondition, LINKS, MaxObjects } from "@local/shared";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../../contexts/session.js";
import { IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { useFocusModes, useFocusModesStore } from "../../../stores/focusModeStore.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { PubSub } from "../../../utils/pubsub.js";
import { SelectorBase } from "../Selector/Selector.js";

type FocusModeOption = Omit<FocusMode, "__typename">;

function getFocusModeOptionDescription(focusMode: FocusModeOption) {
    return focusMode.description;
}

function getFocusModeOptionLabel(focusMode: FocusModeOption) {
    return focusMode.name;
}

function getDisplayIcon() {
    return <IconCommon
        decorative
        name="FocusMode"
    />;
}

/**
 * Sets your focus mode. Can also create new focus modes.
 */
export function FocusModeSelector() {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const putActiveFocusMode = useFocusModesStore(state => state.putActiveFocusMode);
    const focusModeInfo = useFocusModes(session);

    const { canAdd, hasPremium } = useMemo(() => {
        const { hasPremium } = getCurrentUser(session);
        const max = hasPremium ? MaxObjects.FocusMode.User.premium : MaxObjects.FocusMode.User.noPremium;
        return { canAdd: focusModeInfo.all.length < max, hasPremium };
    }, [focusModeInfo.all.length, session]);

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

    const handleChangedFocusMode = useCallback(function handleChangedFocusMode(updatedFocusMode: FocusModeOption) {
        putActiveFocusMode({
            __typename: "ActiveFocusMode" as const,
            focusMode: {
                ...updatedFocusMode,
                __typename: "ActiveFocusModeFocusMode" as const,
            },
            stopCondition: FocusModeStopCondition.NextBegins,
        }, session);
    }, [putActiveFocusMode, session]);

    const addOption = useMemo(function addOptionMemo() {
        return {
            label: t("AddFocusMode"),
            onSelect: handleAddNewFocusMode,
        };
    }, [handleAddNewFocusMode, t]);

    return (
        <SelectorBase
            name="active"
            options={focusModeInfo.all as FocusModeOption[]}
            getDisplayIcon={getDisplayIcon}
            getOptionLabel={getFocusModeOptionLabel}
            getOptionDescription={getFocusModeOptionDescription}
            fullWidth={true}
            inputAriaLabel="Focus Mode"
            label={t("FocusMode", { count: 1, defaultValue: "Focus Mode" })}
            onChange={handleChangedFocusMode}
            addOption={addOption}
            value={(focusModeInfo.active?.focusMode as FocusModeOption | undefined) ?? null}
        />
    );
}
