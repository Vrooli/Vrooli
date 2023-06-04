import { endpointGetProfile, User } from "@local/shared";
import { useCallback, useContext, useEffect, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { SessionContext } from "utils/SessionContext";
import { useDisplayServerError } from "./useDisplayServerError";
import { useLazyFetch } from "./useLazyFetch";

export const useProfileQuery = () => {
    const session = useContext(SessionContext);
    const [getData, { data, loading: isProfileLoading, error }] = useLazyFetch<any, User>(endpointGetProfile);
    useDisplayServerError(error);
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
};
