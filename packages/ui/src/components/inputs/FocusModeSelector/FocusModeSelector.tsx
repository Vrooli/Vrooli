import { FocusModeStopCondition, LINKS, MaxObjects } from "@local/shared";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser, getFocusModeInfo } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { Selector } from "../Selector/Selector";

/**
 * Sets your focus mode. Can also create new focus modes.
 */
export const FocusModeSelector = () => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { active, all } = useMemo(() => getFocusModeInfo(session), [session]);

    const { canAdd, hasPremium } = useMemo(() => {
        const { hasPremium } = getCurrentUser(session);
        const max = hasPremium ? MaxObjects.FocusMode.User.premium : MaxObjects.FocusMode.User.noPremium;
        return { canAdd: all.length < max, hasPremium };
    }, [all.length, session]);

    const handleAddNewFocusMode = () => {
        // If you can add, open settings
        if (canAdd) setLocation(LINKS.SettingsFocusModes);
        // If you can't add and don't have premium, open premium page
        else if (!hasPremium) {
            setLocation(LINKS.Premium);
            PubSub.get().publish("snack", { message: "Upgrade to increase limit", severity: "Info" });
        }
        // Otherwise, show error
        else PubSub.get().publish("snack", { message: "Max reached", severity: "Error" });
    };

    return (
        <>
            {/* Selector */}
            <Formik
                enableReinitialize={true}
                initialValues={{ active: active?.mode ?? "addOption" }}
                // eslint-disable-next-line @typescript-eslint/no-empty-function
                onSubmit={() => { }} // no-op
            >
                <Selector
                    name="active"
                    options={all}
                    getOptionLabel={(r) => r.name}
                    getOptionDescription={(r) => r.description}
                    fullWidth={true}
                    inputAriaLabel="Focus Mode"
                    label={t("FocusMode", { count: 1, defaultValue: "Focus Mode" })}
                    onChange={(newMode) => {
                        newMode && PubSub.get().publish("focusMode", {
                            __typename: "ActiveFocusMode" as const,
                            mode: newMode,
                            stopCondition: FocusModeStopCondition.NextBegins,
                        });
                    }}
                    addOption={{
                        label: t("AddFocusMode"),
                        onSelect: handleAddNewFocusMode,
                    }}
                />
            </Formik>
        </>
    );
};
