import { useState, useCallback } from "react";
import type { ForcedTool } from "./AttachmentButton";

interface UseAttachmentHandlersOptions {
  addAttachment: (file: File, type: "image" | "pdf") => void;
}

export function useAttachmentHandlers({
  addAttachment,
}: UseAttachmentHandlersOptions) {
  const [forcedTool, setForcedTool] = useState<ForcedTool | null>(null);

  const handleImageSelect = useCallback(
    (file: File) => {
      addAttachment(file, "image");
    },
    [addAttachment],
  );

  const handlePDFSelect = useCallback(
    (file: File) => {
      addAttachment(file, "pdf");
    },
    [addAttachment],
  );

  const handleForceTool = useCallback(
    (scenario: string, toolName: string) => {
      setForcedTool({ scenario, toolName });
    },
    [],
  );

  const handleClearForcedTool = useCallback(() => {
    setForcedTool(null);
  }, []);

  return {
    forcedTool,
    setForcedTool,
    handleImageSelect,
    handlePDFSelect,
    handleForceTool,
    handleClearForcedTool,
  };
}
