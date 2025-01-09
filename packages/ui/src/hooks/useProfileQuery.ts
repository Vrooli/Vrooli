import { endpointsUser, User } from "@local/shared";
import { SessionContext } from "contexts";
import { useCallback, useContext, useEffect, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { useLazyFetch } from "./useLazyFetch";

export function useProfileQuery() {
    const session = useContext(SessionContext);
    const [getData, { data, loading: isProfileLoading }] = useLazyFetch<never, User>(endpointsUser.profile);
    useEffect(() => {
        if (getCurrentUser(session).id) getData();
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
