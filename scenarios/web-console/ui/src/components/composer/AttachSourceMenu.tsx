import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Camera, FolderOpen, Monitor, Smartphone, Tablet, Upload } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ContextMenu, type ContextMenuItem } from "@vrooli/react-component-library/ContextMenu/1";

import CameraCaptureDialog from "./CameraCaptureDialog";
import HostFilePicker from "./HostFilePicker";
import { strings } from "../../consts/strings";
import type { HostAttachmentPick } from "../../hooks/useComposerAttachments";
import {
  CAMERA_PICKER_ACCEPT,
  cameraCaptureSupported,
  isLoopbackHostname,
  nativeChooserPlatform,
  planAttachSources,
  readAttachEnvironment,
  type AttachEnvironment,
  type AttachSourceId,
} from "../../lib/attachments";

interface AttachSourceMenuProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Receives files chosen on this device (native picker, file dialog, camera). */
  onFilesPicked: (files: File[]) => void;
  /**
   * Session whose machine the host source browses. With onHostPathsPicked it
   * adds the host source; without either, the menu offers this device only.
   */
  hostSessionId?: string;
  /** Receives host files/folders picked by path (nothing is uploaded). */
  onHostPathsPicked?: (picks: HostAttachmentPick[]) => void;
  /** Anchor the popover to a trigger element (preferred). */
  anchorRef?: React.RefObject<HTMLElement | null>;
  /** Or place it at viewport coordinates (e.g. from a click event). */
  position?: { x: number; y: number };
  /** Prefix for the trigger/input test ids so multiple surfaces do not collide. */
  testIdPrefix?: string;
  /**
   * Whether the owning surface is currently open. The camera and host picker
   * are torn down when it is not — but NOT when the source menu merely closes,
   * since choosing a source closes the menu in order to open it.
   */
  active?: boolean;
  /** Test seam: the browser environment the source plan reads. */
  environment?: AttachEnvironment;
}

interface MediaDevicesEnumerator {
  enumerateDevices?: () => Promise<MediaDeviceInfo[]>;
}

/**
 * useCameraPresent reports whether the in-app camera is worth offering: the
 * browser can open a stream and, where it can list devices, one is a camera.
 * Device kinds are listed without a permission prompt, so this never asks.
 */
function useCameraPresent(open: boolean): boolean {
  const [present, setPresent] = useState(cameraCaptureSupported);
  useEffect(() => {
    if (!open || !cameraCaptureSupported()) return;
    const devices = (globalThis.navigator as unknown as { mediaDevices?: MediaDevicesEnumerator }).mediaDevices;
    if (!devices?.enumerateDevices) return;
    let cancelled = false;
    devices.enumerateDevices().then(
      (list) => { if (!cancelled) setPresent(list.some((d) => d.kind === "videoinput")); },
      () => { /* keep the optimistic default */ },
    );
    return () => { cancelled = true; };
  }, [open]);
  return present;
}

/**
 * AttachSourceMenu — the shared attach menu used by both the full-screen
 * composer and the touch toolbar. It asks WHICH MACHINE a file comes from and
 * leaves camera / library / files to the device's own picker wherever the OS
 * has one (iOS, iPadOS, Android). Desktop browsers have no such chooser, so
 * there it offers the file dialog and the in-app camera. The host source walks
 * the session's machine and attaches by path. The menu owns its hidden inputs,
 * the camera dialog, and the host picker; the parent owns the trigger and what
 * happens to the picks.
 */
export default function AttachSourceMenu({
  open,
  onOpenChange,
  onFilesPicked,
  hostSessionId,
  onHostPathsPicked,
  anchorRef,
  position,
  testIdPrefix = "composer",
  active = true,
  environment,
}: AttachSourceMenuProps) {
  const { t } = useTranslation();
  const filesInputRef = useRef<HTMLInputElement>(null);
  const cameraInputRef = useRef<HTMLInputElement>(null);
  const [cameraOpen, setCameraOpen] = useState(false);
  const [hostOpen, setHostOpen] = useState(false);
  const cameraPresent = useCameraPresent(open);
  const env = useMemo(() => environment ?? readAttachEnvironment(), [environment]);
  const hostAvailable = Boolean(hostSessionId && onHostPathsPicked);
  const sources = useMemo(
    () => planAttachSources(env, { hostAvailable, cameraAvailable: cameraPresent }),
    [env, hostAvailable, cameraPresent],
  );

  // Take the camera and host picker down only when the owning surface closes.
  // The menu closing is NOT a reason: picking a source closes the menu first.
  useEffect(() => {
    if (!active) {
      setCameraOpen(false);
      setHostOpen(false);
    }
  }, [active]);

  const handlePicked = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(event.target.files ?? []);
      event.target.value = "";
      if (files.length > 0) onFilesPicked(files);
    },
    [onFilesPicked],
  );

  // Inputs are clicked synchronously inside the menu's select handler: iOS
  // only opens a file picker from within the user's own tap.
  const items = useMemo<ContextMenuItem[]>(() => {
    const onHost = isLoopbackHostname(env.hostname);
    const platform = nativeChooserPlatform(env);
    const build: Record<AttachSourceId, () => ContextMenuItem> = {
      device: () => ({
        id: "device",
        label: t(platform === "iphone" ? strings.composer.attachDeviceIphone : platform === "ipad" ? strings.composer.attachDeviceIpad : strings.composer.attachDevicePhone),
        icon: platform === "ipad" ? <Tablet className="h-4 w-4 shrink-0" /> : <Smartphone className="h-4 w-4 shrink-0" />,
        testId: `${testIdPrefix}-attach-device`,
        onSelect: () => { onOpenChange(false); filesInputRef.current?.click(); },
      }),
      upload: () => ({
        id: "upload",
        label: t(onHost ? strings.composer.attachUploadCopy : strings.composer.attachUpload),
        icon: <Upload className="h-4 w-4 shrink-0" />,
        testId: `${testIdPrefix}-attach-upload`,
        onSelect: () => { onOpenChange(false); filesInputRef.current?.click(); },
      }),
      camera: () => ({
        id: "camera",
        label: t(strings.composer.attachCamera),
        icon: <Camera className="h-4 w-4 shrink-0" />,
        testId: `${testIdPrefix}-attach-camera`,
        onSelect: () => {
          onOpenChange(false);
          if (cameraCaptureSupported()) setCameraOpen(true);
          else cameraInputRef.current?.click();
        },
      }),
      host: () => ({
        id: "host",
        label: t(onHost ? strings.composer.attachHostBrowse : strings.composer.attachHost),
        icon: onHost ? <FolderOpen className="h-4 w-4 shrink-0" /> : <Monitor className="h-4 w-4 shrink-0" />,
        testId: `${testIdPrefix}-attach-host`,
        onSelect: () => { onOpenChange(false); setHostOpen(true); },
      }),
    };
    return sources.map((id) => build[id]());
  }, [env, onOpenChange, sources, t, testIdPrefix]);

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
        items={items}
      />
      {/* No accept/capture: a plain multi-file input is what makes iOS and
          Android show their full camera + library + files chooser. */}
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
      {hostSessionId && onHostPathsPicked && (
        <HostFilePicker
          open={hostOpen}
          sessionId={hostSessionId}
          onClose={() => { setHostOpen(false); }}
          onAttach={onHostPathsPicked}
        />
      )}
    </>
  );
}
