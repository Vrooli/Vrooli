import { ActiveFocusMode, FocusMode, FocusModeSearchInput, FocusModeSearchResult, FocusModeStopCondition, LINKS, MaxObjects, Session, SetActiveFocusModeInput, endpointsFocusMode } from "@local/shared";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { create } from "zustand";
import { fetchData } from "../../../api/fetchData.js";
import { ServerResponseParser } from "../../../api/responseParser.js";
import { SessionContext } from "../../../contexts.js";
import { IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { PubSub } from "../../../utils/pubsub.js";
import { SelectorBase } from "../Selector/Selector.js";

type FocusModeOption = Omit<FocusMode, "__typename">;

export type FocusModeInfo = {
    active: ActiveFocusMode | null;
    all: FocusMode[];
};

interface FocusModesState {
    focusModes: FocusMode[];
    isLoading: boolean;
    /** Fetch focus modes from server */
    fetchFocusModes: (signal?: AbortSignal) => Promise<FocusMode[]>;
    /** Get focus mode info (including active focus mode) */
    getFocusModeInfo: (session: Session | null | undefined, signal?: AbortSignal) => Promise<FocusModeInfo>;
    /** Update focus mode in store */
    setFocusMode: (focusMode: FocusMode) => unknown;
    /** Update all focus modes in store */
    setFocusModes: (focusModes: FocusMode[] | ((previousFocusModes: FocusMode[]) => FocusMode[])) => unknown;
    /** Update active focus mode in both store and database */
    putActiveFocusMode: (activeFocusMode: ActiveFocusMode | null, session: Session | null | undefined, signal?: AbortSignal) => Promise<unknown>;
}

export const useFocusModesStore = create<FocusModesState>()((set, get) => ({
    focusModes: [],
    isLoading: false,
    fetchFocusModes: async (signal) => {
        if (get().focusModes.length > 0 || get().isLoading) {
            // Already fetched or currently fetching
            return get().focusModes;
        }

        set({ isLoading: true });

        try {
            const response = await fetchData<FocusModeSearchInput, FocusModeSearchResult>({
                ...endpointsFocusMode.findMany,
                inputs: {},
                signal,
            });

            if (response.errors) {
                ServerResponseParser.displayErrors(response.errors);
                throw new Error("Failed to fetch focus modes");
            }

            const focusModes = response.data?.edges.map(edge => edge.node) ?? [];

            set({ focusModes, isLoading: false });
            return focusModes;
        } catch (error) {
            if ((error as { name?: string }).name === "AbortError") return [];
            console.error("Error fetching focus modes:", error);
            set({ isLoading: false });
            return [];
        }
    },
    // For some reason, the typing of useFocusModesStore breaks unless we explicitly add types here
    getFocusModeInfo: async (session: Session | null | undefined, signal?: AbortSignal): Promise<FocusModeInfo> => {
        // If not logged in, return empty
        if (!session || session.isLoggedIn === false) {
            return { active: null, all: [] };
        }

        // Fetch focus modes
        let focusModes = await useFocusModesStore.getState().fetchFocusModes(signal);
        // Check session for active focus mode
        let activeFocusMode = getCurrentUser(session).activeFocusMode ?? null;
        const fullActiveFocusModeInfo = focusModes.find(focusMode => focusMode.id === activeFocusMode?.focusMode?.id);
        if (activeFocusMode && fullActiveFocusModeInfo) {
            activeFocusMode = {
                ...activeFocusMode,
                focusMode: {
                    ...fullActiveFocusModeInfo,
                    __typename: "ActiveFocusModeFocusMode" as const,
                },
            };
        }

        // If there is an active focus mode, move it to the first position in the 'all' array
        if (fullActiveFocusModeInfo) {
            focusModes = focusModes.filter(focusMode => focusMode.id !== fullActiveFocusModeInfo.id);
            focusModes.unshift(fullActiveFocusModeInfo);
        }

        return { active: activeFocusMode, all: focusModes };
    },
    setFocusMode: (focusMode) => {
        const focusModes = get().focusModes;
        const index = focusModes.findIndex(f => f.id === focusMode.id);
        if (index === -1) return;
        focusModes[index] = focusMode;
        set({ focusModes });
    },
    setFocusModes: (focusModes) => {
        set((state) => {
            const newFocusModes =
                typeof focusModes === "function"
                    ? focusModes(state.focusModes)
                    : focusModes;
            return { focusModes: newFocusModes };
        });
    },
    putActiveFocusMode: async (activeFocusMode, session, signal) => {
        // Update active focus mode in database
        try {
            const response = await fetchData<SetActiveFocusModeInput, ActiveFocusMode>({
                ...endpointsFocusMode.setActive,
                inputs: {
                    id: activeFocusMode?.focusMode.id,
                    stopCondition: activeFocusMode?.stopCondition,
                    stopTime: activeFocusMode?.stopTime,
                },
                signal,
            });

            if (response.errors) {
                ServerResponseParser.displayErrors(response.errors);
                throw new Error("Failed to fetch focus modes");
            }
            // Update active focus mode in UI session data
            else if (session) {
                const updatedSession = { ...session };
                updatedSession.users = updatedSession.users?.map((user, index) => {
                    if (index === 0) {
                        return {
                            ...user,
                            activeFocusMode: response.data,
                        };
                    }
                    return user;
                });
                PubSub.get().publish("session", updatedSession);
            }
        } catch (error) {
            if ((error as { name?: string }).name === "AbortError") return [];
            console.error("Error fetching focus modes:", error);
            return [];
        }
    },
}));

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

    const getFocusModeInfo = useFocusModesStore(state => state.getFocusModeInfo);
    const putActiveFocusMode = useFocusModesStore(state => state.putActiveFocusMode);
    const [focusModeInfo, setFocusModeInfo] = useState<FocusModeInfo>({ active: null, all: [] });
    useEffect(function fetchFocusModeInfoEffect() {
        const abortController = new AbortController();

        async function fetchFocusModeInfo() {
            const info = await getFocusModeInfo(session, abortController.signal);
            setFocusModeInfo(info);
        }

        fetchFocusModeInfo();

        return () => {
            abortController.abort();
        };
    }, [getFocusModeInfo, session]);

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
