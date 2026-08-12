import { useCallback, useEffect, useMemo, useState } from "react";
import { getTTSSummarizeConfig, listTTSSummarizeModels, updateTTSSummarizeConfig, } from "../../audio-integration";
import { toErrorInfo } from "../../lib/errors";
export function useSummarizeSettings() {
    const [config, setConfig] = useState(null);
    const [models, setModels] = useState([]);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState(null);
    const selectedModel = useMemo(() => models.find((model) => model.id === config?.model) ?? null, [config?.model, models]);
    const load = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const [nextConfig, nextModels] = await Promise.all([
                getTTSSummarizeConfig(),
                listTTSSummarizeModels(),
            ]);
            setConfig(nextConfig);
            setModels(nextModels);
        }
        catch (loadError) {
            setError(toErrorInfo(loadError).message);
        }
        finally {
            setLoading(false);
        }
    }, []);
    useEffect(() => {
        void load();
    }, [load]);
    const save = useCallback(async (patch) => {
        setSaving(true);
        setError(null);
        setConfig((prev) => prev ? { ...prev, ...patch } : prev);
        try {
            const updated = await updateTTSSummarizeConfig(patch);
            setConfig(updated);
            return updated;
        }
        catch (saveError) {
            setError(toErrorInfo(saveError).message);
            await load();
            return null;
        }
        finally {
            setSaving(false);
        }
    }, [load]);
    return {
        config,
        models,
        selectedModel,
        loading,
        saving,
        error,
        setConfig,
        load,
        save,
    };
}
