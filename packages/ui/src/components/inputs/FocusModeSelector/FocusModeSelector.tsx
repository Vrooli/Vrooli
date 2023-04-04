import { FocusModeStopCondition, LINKS } from "@shared/consts";
import { useLocation } from "@shared/route";
import { Formik } from "formik";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getFocusModeInfo } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { Selector } from "../Selector/Selector";

/**
 * Sets your focus mode. Can also create new focus modes.
 */
export const FocusModeSelector = () => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { active, all } = useMemo(() => getFocusModeInfo(session), [session]);
    console.log('focus mode selector', active, all);

    const handleAddNewFocusMode = () => {
        setLocation(LINKS.SettingsFocusModes)
    }

    return (
        <>
            {/* Selector */}
            <Formik
                enableReinitialize={true}
                initialValues={{ active }}
                onSubmit={() => { }} // no-op
            >
                <Selector
                    name="active"
                    options={all}
                    getOptionLabel={(r) => r.name}
                    getOptionDescription={(r) => r.description}
                    fullWidth={true}
                    inputAriaLabel="Focus Mode"
                    label={t('FocusMode', { count: 1, defaultValue: 'Focus Mode' })}
                    onChange={(newMode) => {
                        newMode && PubSub.get().publishFocusMode({
                            __typename: 'ActiveFocusMode' as const,
                            mode: newMode,
                            stopCondition: FocusModeStopCondition.Automatic,
                        })
                    }}
                    addOption={{
                        label: t("AddFocusMode"),
                        onSelect: handleAddNewFocusMode,
                    }}
                />
            </Formik>
        </>
    )
}