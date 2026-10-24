import { useCallback, useEffect, useRef, useState } from "react";
import { Camera, Images, Paperclip } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ContextMenu } from "@vrooli/react-component-library/ContextMenu/1";

import CameraCaptureDialog from "./CameraCaptureDialog";
import { strings } from "../../consts/strings";
import { CAMERA_PICKER_ACCEPT, MEDIA_PICKER_ACCEPT, cameraCaptureSupported } from "../../lib/attachments";

interface AttachSourceMenuProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Receives files chosen from any source (camera, photos, or file picker). */
  onFilesPicked: (files: File[]) => void;
  /** Anchor the popover to a trigger element (preferred). */
  anchorRef?: React.RefObject<HTMLElement | null>;
  /** Or place it at viewport coordinates (e.g. from a click event). */
  position?: { x: number; y: number };
  /** Prefix for the trigger/input test ids so multiple surfaces do not collide. */
  testIdPrefix?: string;
  /**
   * Whether the owning surface is currently open. The camera is torn down when
   * it is not — but NOT when the source menu merely closes, since choosing
   * Camera closes the menu in order to open the dialog.
   */
  active?: boolean;
}

/**
 * AttachSourceMenu — the shared Camera / Photos / Files source picker used by
 * both the full-screen composer and the touch toolbar. It owns its hidden file
 * inputs and the in-app camera dialog; the parent owns the trigger button and
 * decides what to do with the picked files (stage them, open the composer, …).
 */
export default function AttachSourceMenu({
  open,
  onOpenChange,
  onFilesPicked,
  anchorRef,
  position,
  testIdPrefix = "composer",
  active = true,
}: AttachSourceMenuProps) {
  const { t } = useTranslation();
  const photosInputRef = useRef<HTMLInputElement>(null);
  const filesInputRef = useRef<HTMLInputElement>(null);
  const cameraInputRef = useRef<HTMLInputElement>(null);
  const [cameraOpen, setCameraOpen] = useState(false);

  // Take the camera down only when the owning surface closes. The menu closing
  // is NOT a reason to stop the camera: picking Camera closes the menu first.
  useEffect(() => {
    if (!active) setCameraOpen(false);
  }, [active]);

  const handlePicked = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(event.target.files ?? []);
      event.target.value = "";
      if (files.length > 0) onFilesPicked(files);
    },
    [onFilesPicked],
  );

  const handleCamera = useCallback(() => {
    onOpenChange(false);
    if (cameraCaptureSupported()) setCameraOpen(true);
    else cameraInputRef.current?.click();
  }, [onOpenChange]);

  return (
    <>
      <ContextMenu
        open={open}
        onOpenChange={onOpenChange}
        anchorRef={anchorRef}
        position={position}
        placement="bottom-start"
        title={t(strings.composer.attachTitle)}
        closeLabel={t(strings.composer.attachTitle)}
        testId={`${testIdPrefix}-attach-menu`}
        items={[
          {
            id: "camera",
            label: t(strings.composer.attachCamera),
            icon: <Camera className="h-4 w-4 shrink-0" />,
            testId: `${testIdPrefix}-attach-camera`,
            onSelect: handleCamera,
          },
          {
            id: "photos",
            label: t(strings.composer.attachPhotos),
            icon: <Images className="h-4 w-4 shrink-0" />,
            testId: `${testIdPrefix}-attach-photos`,
            onSelect: () => { onOpenChange(false); photosInputRef.current?.click(); },
          },
          {
            id: "files",
            label: t(strings.composer.attachFiles),
            icon: <Paperclip className="h-4 w-4 shrink-0" />,
            testId: `${testIdPrefix}-attach-files`,
            onSelect: () => { onOpenChange(false); filesInputRef.current?.click(); },
          },
        ]}
      />
      <input
        ref={photosInputRef}
        type="file"
        accept={MEDIA_PICKER_ACCEPT}
        multiple
        hidden
        data-testid={`${testIdPrefix}-photos-input`}
        onChange={handlePicked}
      />
      <input
        ref={filesInputRef}
        type="file"
        multiple
        hidden
        data-testid={`${testIdPrefix}-file-input`}
        onChange={handlePicked}
      />
      <input
        ref={cameraInputRef}
        type="file"
        accept={CAMERA_PICKER_ACCEPT}
        capture="environment"
        hidden
        data-testid={`${testIdPrefix}-camera-input`}
        onChange={handlePicked}
      />
      <CameraCaptureDialog
        open={cameraOpen}
        onClose={() => { setCameraOpen(false); }}
        onCapture={(file) => { onFilesPicked([file]); }}
        onUseFilePicker={() => { cameraInputRef.current?.click(); }}
      />
    </>
  );
}
