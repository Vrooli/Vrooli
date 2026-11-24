import { useCallback, useEffect, useRef, useState } from "react";

import {
  deleteWorkspaceAsset,
  listWorkspaceAssets,
  uploadWorkspaceAsset,
  type WorkspaceAsset,
  type WorkspaceAssetFidelity,
} from "../../api/workspaces";
import { assertImageUpload } from "./assets";

interface WorkspaceAssets {
  assets: WorkspaceAsset[];
  assetsLoading: boolean;
  assetError: string | null;
  importAsset: (file: File, fidelity?: WorkspaceAssetFidelity) => Promise<void>;
  removeAsset: (assetId: string) => Promise<void>;
  reloadAssets: () => Promise<void>;
}

/** Owns asset requests for the currently selected workspace. */
export function useWorkspaceAssets(workspaceId: string): WorkspaceAssets {
  const [assets, setAssets] = useState<WorkspaceAsset[]>([]);
  const [assetsLoading, setAssetsLoading] = useState(false);
  const [assetError, setAssetError] = useState<string | null>(null);
  const generation = useRef(0);
  const mounted = useRef(false);
  const workspaceIdRef = useRef(workspaceId);
  workspaceIdRef.current = workspaceId;

  const reloadAssets = useCallback(async () => {
    if (!mounted.current) return;
    const owner = workspaceId;
    const request = ++generation.current;
    setAssetsLoading(true);
    try {
      const list = await listWorkspaceAssets(owner);
      if (generation.current === request && workspaceIdRef.current === owner) {
        setAssets(list);
        setAssetError(null);
      }
    } catch (cause) {
      if (generation.current === request && workspaceIdRef.current === owner) {
        setAssetError(cause instanceof Error ? cause.message : "assets-unavailable");
      }
      throw cause;
    } finally {
      if (generation.current === request && workspaceIdRef.current === owner) {
        setAssetsLoading(false);
      }
    }
  }, [workspaceId]);

  useEffect(() => {
    mounted.current = true;
    setAssets([]);
    setAssetError(null);
    void reloadAssets().catch(() => undefined);
    return () => {
      // Ignore both pending list calls and mutations whose workspace view has
      // been left. This also prevents late state writes after unmount.
      generation.current += 1;
      mounted.current = false;
    };
  }, [workspaceId, reloadAssets]);

  const importAsset = useCallback(
    async (file: File, fidelity: WorkspaceAssetFidelity = "reference") => {
      assertImageUpload(file);
      const owner = workspaceId;
      const asset = await uploadWorkspaceAsset(owner, file, fidelity);
      if (!mounted.current || workspaceIdRef.current !== owner) return;
      generation.current += 1;
      setAssets((current) => [asset, ...current.filter((entry) => entry.id !== asset.id)]);
      setAssetError(null);
    },
    [workspaceId],
  );

  const removeAsset = useCallback(
    async (assetId: string) => {
      const owner = workspaceId;
      await deleteWorkspaceAsset(owner, assetId);
      if (!mounted.current || workspaceIdRef.current !== owner) return;
      generation.current += 1;
      setAssets((current) => current.filter((entry) => entry.id !== assetId));
    },
    [workspaceId],
  );

  return { assets, assetsLoading, assetError, importAsset, removeAsset, reloadAssets };
}
