import { endpointsUser, User } from "@local/shared";
import { useCallback, useContext, useEffect, useState } from "react";
import { getCurrentUser } from "utils/authentication/session.js";
import { SessionContext } from "../contexts.js";
import { useLazyFetch } from "./useLazyFetch.js";

export function useProfileQuery() {
    const session = useContext(SessionContext);
    const [getData, { data, loading: isProfileLoading }] = useLazyFetch<never, User>(endpointsUser.profile);
    useEffect(() => {
        if (getCurrentUser(session).id) {
            getData();
        } else {
            console.warn("No user ID found in session. Cannot fetch profile.");
        }
    }, [getData, session]);
    const [profile, setProfile] = useState<User | undefined>(undefined);
    useEffect(() => {
        if (data) setProfile(data);
    }, [data]);
    const onProfileUpdate = useCallback((updatedProfile: User | undefined) => {
        if (updatedProfile) setProfile(updatedProfile);
    }, []);

    return {
        onProfileUpdate,
        isProfileLoading,
        profile,
    };
}
