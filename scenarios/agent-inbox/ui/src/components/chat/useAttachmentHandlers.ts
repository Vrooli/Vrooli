import { useCallback } from "react";

interface UseAttachmentHandlersOptions {
  addAttachment: (file: File, type: "image" | "pdf") => void;
}

export function useAttachmentHandlers({
  addAttachment,
}: UseAttachmentHandlersOptions) {
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

  return {
    handleImageSelect,
    handlePDFSelect,
  };
}
