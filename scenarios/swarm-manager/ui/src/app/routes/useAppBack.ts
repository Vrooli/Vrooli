import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { graphPath } from "./route-paths";

const DEFAULT_FALLBACK = graphPath({ lens: "plan" });

function hasInAppHistory(): boolean {
  const state = window.history.state as { idx?: unknown } | null;
  return typeof state?.idx === "number" && state.idx > 0;
}

export function useAppBack(fallback: string = DEFAULT_FALLBACK): () => void {
  const navigate = useNavigate();
  return useCallback(() => {
    if (hasInAppHistory()) {
      navigate(-1);
      return;
    }
    navigate(fallback, { replace: true });
  }, [fallback, navigate]);
}

export function getDefaultBackFallback(): string {
  return DEFAULT_FALLBACK;
}
