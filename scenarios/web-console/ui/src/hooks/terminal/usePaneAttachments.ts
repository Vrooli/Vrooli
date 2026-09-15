import { useCallback, useRef, useState, type ChangeEvent, type ClipboardEvent, type DragEvent } from "react";
import { useImageUpload } from "../useImageUpload";
import type { GateResult, InputIntent } from "../../components/terminal/inputGate";

export function usePaneAttachments(sessionId: string, submitInput: (data: string, intent: Exclude<InputIntent, "control">) => GateResult, closeContextMenu: () => void) {
  const { uploadAndInject, uploading, error: uploadError } = useImageUpload(sessionId, submitInput);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [dragOver, setDragOver] = useState(false);
  const handleCtxUploadImage = useCallback(() => {
    closeContextMenu();
    fileInputRef.current?.click();
  }, [closeContextMenu]);
  const handleFileInputChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) void uploadAndInject(file);
    event.target.value = "";
  }, [uploadAndInject]);
  const handlePaste = useCallback((event: ClipboardEvent) => {
    const items = event.clipboardData.items;
    for (const item of items) {
      if (!item.type.startsWith("image/")) continue;
      event.preventDefault();
      const blob = item.getAsFile();
      if (blob) void uploadAndInject(blob);
      return;
    }
  }, [uploadAndInject]);
  const handleDragOver = useCallback((event: DragEvent) => {
    event.preventDefault();
    setDragOver(true);
  }, []);
  const handleDragLeave = useCallback(() => { setDragOver(false); }, []);
  const handleDrop = useCallback((event: DragEvent) => {
    event.preventDefault();
    setDragOver(false);
    for (const file of Array.from(event.dataTransfer.files)) {
      if (file.type.startsWith("image/")) void uploadAndInject(file);
    }
  }, [uploadAndInject]);
  return { fileInputRef, dragOver, uploading, uploadError, handleCtxUploadImage, handleFileInputChange, handlePaste, handleDragOver, handleDragLeave, handleDrop };
}
