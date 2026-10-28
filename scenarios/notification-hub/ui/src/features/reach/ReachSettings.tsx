import { Button } from "@vrooli/react-component-library/Button/2";
import { FormField } from "@vrooli/react-component-library/FormField/1.0.4";
import { RefreshCw } from "lucide-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { getInjectedConfig } from "@vrooli/api-base";
import { recipientsClient, registerBrowserPushSubscription } from "../../api/notifications";
import { forceServiceWorkerUpdate, readConnectionStatus, type ConnectionStatus } from "../../api/serviceWorker";
import { SessionPanel } from "../session/SessionGate";
import { Link } from "react-router-dom";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { useTranslation } from "../../i18n";
import type { TFunction } from "i18next";
import { useQuietEditorHeader } from "../../layout/AppShell";

const LABELS = ["public", "private", "sensitive", "critical"] as const;
type ReachDevice = Awaited<ReturnType<typeof recipientsClient.listDevices>>["devices"][number];

export function ReachSettings({ signedIn, focusedEntry = false, preferencesHref = "/settings?view=preferences" }: { signedIn: boolean; focusedEntry?: boolean; preferencesHref?: string }) {
  const { t } = useTranslation();
  const setContextHeader = useQuietEditorHeader();
  const queryClient = useQueryClient();
  const [preflight, setPreflight] = useState(false);
  const [requestingPermission, setRequestingPermission] = useState(false);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [permissionRecoveryDismissed, setPermissionRecoveryDismissed] = useState(false);
  const [enrolling, setEnrolling] = useState(false);
  const [pushMessage, setPushMessage] = useState("");
  const [removeId, setRemoveId] = useState<string | null>(null);
  const [removalResult, setRemovalResult] = useState<{ device: ReachDevice; remaining: ReachDevice[]; responseUnknown: boolean } | null>(null);
  const [selectedDeviceId, setSelectedDeviceId] = useState("");
  const [channelReadGapFocused, setChannelReadGapFocused] = useState(false);
  const [channelDeviceId, setChannelDeviceId] = useState<string | null>(null);
  const [editingChannel, setEditingChannel] = useState("email");
  const [destination, setDestination] = useState("");
  const [labels, setLabels] = useState<string[]>(["public"]);
  const [channelAddressId, setChannelAddressId] = useState<string | null>(null);
  const [createAddressId, setCreateAddressId] = useState<string | null>(null);
  const [channelSaveMessage, setChannelSaveMessage] = useState("");
  const [channelSaveError, setChannelSaveError] = useState("");
  const [savingChannel, setSavingChannel] = useState(false);
  const [reviewChannel, setReviewChannel] = useState(false);
  const [confirmDiscard, setConfirmDiscard] = useState(false);
  const discardTriggerRef = useRef<HTMLButtonElement | null>(null);
  const channelEntryTriggerRef = useRef<HTMLButtonElement | null>(null);
  const channelEntryTriggerIdRef = useRef<string | null>(null);
  const discardReviewHeadingRef = useRef<HTMLHeadingElement | null>(null);
  const channelReadGapTriggerIdRef = useRef<string | null>(null);
  const discardTransitionRef = useRef<"open" | "keep" | "discard" | null>(null);
  const removeTriggerIdRef = useRef<string | null>(null);
  const removeReviewHeadingRef = useRef<HTMLHeadingElement | null>(null);
  const restoreRemoveFocusRef = useRef(false);
  const removalConfirmedRef = useRef(false);
  // Device ownership and registration can change while Settings is mounted,
  // including after a route handoff. Never present cached rows as the latest
  // owner-scoped device list when Reach is entered again.
  const devices = useQuery({ queryKey: ["recipient-devices"], queryFn: () => recipientsClient.listDevices({}), refetchOnWindowFocus: true, refetchOnMount: "always" });
  const channelAddresses = useQuery({ queryKey: ["recipient-channel-addresses"], queryFn: () => recipientsClient.listChannelAddresses({}), refetchOnWindowFocus: true, enabled: signedIn });
  const status = useQuery({ queryKey: ["connection-status"], queryFn: readConnectionStatus, refetchOnWindowFocus: true });
  const update = useMutation({ mutationFn: forceServiceWorkerUpdate, onSuccess: () => void status.refetch() });
  const remove = useMutation({
    mutationFn: async ({ id, device }: { id: string; device: ReachDevice }) => {
      let requestError: unknown;
      try {
        await recipientsClient.removeDevice({ id });
      } catch (error) {
        requestError = error;
      }
      await queryClient.cancelQueries({ queryKey: ["recipient-devices"], exact: true });
      const latest = await queryClient.fetchQuery({ queryKey: ["recipient-devices"], queryFn: () => recipientsClient.listDevices({}), staleTime: 0 });
      if (!latest.devices.some((row) => row.id === id)) return { device, remaining: latest.devices, responseUnknown: Boolean(requestError) };
      if (requestError) throw requestError;
      throw new Error("Device remains in the latest owner-scoped read");
    },
    onSuccess: (result) => {
      removalConfirmedRef.current = true;
      setRemoveId(null);
      setRemovalResult(result);
      setPushMessage(result.responseUnknown
        ? t(strings.reach.copy.deviceRemoveUncertain)
        : t(strings.reach.copy.deviceRemoved));
    },
    onError: () => setPushMessage(t(strings.reach.copy.deviceRemoveUnknown)),
    onSettled: () => {
      void devices.refetch();
      void queryClient.invalidateQueries({ queryKey: ["recipient-channel-addresses"] });
    },
  });

  const enableBrowser = async () => {
    setPreflight(false);
    setPushMessage("");
    let registrationRequestAccepted = false;
    const browser = status.data;
    if (!browser?.supported) { setPushMessage(t(strings.reach.copy.unsupportedPush)); return; }
    if (browser.permission === "denied") { setPushMessage(t(strings.reach.copy.permissionDeniedEnroll)); return; }
    try {
      if (browser.permission === "default") {
        if (typeof Notification === "undefined") { setPushMessage(t(strings.reach.copy.permissionControlsMissing)); return; }
        setRequestingPermission(true);
        let permission: NotificationPermission;
        try {
          permission = await Notification.requestPermission();
        } catch {
          setPushMessage(t(strings.reach.copy.permissionReadUnavailable));
          return;
        } finally {
          setRequestingPermission(false);
        }
        if (permission !== "granted") {
          if (permission === "denied") {
            setPermissionDenied(true);
            setPermissionRecoveryDismissed(true);
          }
          setPushMessage(permission === "denied" ? t(strings.reach.copy.permissionDeniedResult) : t(strings.reach.copy.permissionPending));
          await status.refetch();
          return;
        }
      }
      let key = devices.data?.vapidPublicKey;
      if (!key) key = (await recipientsClient.listDevices({})).vapidPublicKey;
      key = key || getInjectedConfig()?.VAPID_PUBLIC_KEY as string | undefined || import.meta.env.VITE_VAPID_PUBLIC_KEY as string | undefined;
      if (!key) { setPushMessage(t(strings.reach.copy.publicKeyMissing)); return; }
      // The endpoint currently grants all four labels. Never show a scope picker
      // that the server contract cannot enforce.
      setEnrolling(true);
      await registerBrowserPushSubscription(base64UrlToBytes(key) as unknown as BufferSource);
      registrationRequestAccepted = true;
      await queryClient.cancelQueries({ queryKey: ["recipient-devices"], exact: true });
      const latestDevices = await queryClient.fetchQuery({ queryKey: ["recipient-devices"], queryFn: () => recipientsClient.listDevices({}), staleTime: 0 });
      const registeredHere = latestDevices.devices.some((device) => device.name === window.location.origin && device.channels.includes("web_push"));
      if (!registeredHere) {
        setPushMessage(t(strings.reach.copy.registrationUnknown));
        await status.refetch();
        return;
      }
      await queryClient.cancelQueries({ queryKey: ["connection-status"], exact: true });
      const connection = await queryClient.fetchQuery<ConnectionStatus>({ queryKey: ["connection-status"], queryFn: readConnectionStatus, staleTime: 0 });
      setPushMessage(connection.subscribed
        ? t(strings.reach.copy.registrationConfirmedSubscribed)
        : t(strings.reach.copy.registrationConfirmedNoSubscription));
    } catch {
      const refreshed = await devices.refetch().catch(() => undefined);
      const ownDevice = refreshed?.data?.devices.find((device) => device.name === window.location.origin && device.channels.includes("web_push"));
      setPushMessage(registrationRequestAccepted
        ? ownDevice
          ? t(strings.reach.copy.registrationRequestConfirmed)
          : t(strings.reach.copy.registrationRequestUnknown)
        : ownDevice
          ? t(strings.reach.copy.registrationConflict)
          : t(strings.reach.copy.registrationNotConfirmed));
      await status.refetch().catch(() => undefined);
    } finally {
      setEnrolling(false);
    }
  };

  const openEmail = (deviceId: string, address?: { id: string; address: string; approvedLabels: string[] }, trigger?: HTMLButtonElement) => {
    channelEntryTriggerRef.current = trigger ?? null;
    channelEntryTriggerIdRef.current = trigger?.id || null;
    setEditingChannel("email"); setChannelDeviceId(deviceId); setChannelAddressId(address?.id ?? null); setCreateAddressId(null); setDestination(address?.address ?? ""); setLabels(address?.approvedLabels ?? ["public"]); setReviewChannel(false); setConfirmDiscard(false); setChannelSaveError("");
  };
  const openExistingChannel = (deviceId: string, address: { id: string; channel: string; address: string; approvedLabels: string[] }, trigger?: HTMLButtonElement) => {
    channelEntryTriggerRef.current = trigger ?? null;
    channelEntryTriggerIdRef.current = trigger?.id || null;
    setEditingChannel(address.channel); setChannelDeviceId(deviceId); setChannelAddressId(address.id); setCreateAddressId(null); setDestination(address.address); setLabels(address.approvedLabels); setReviewChannel(false); setConfirmDiscard(false); setChannelSaveError("");
  };
  const requestDiscard = () => { discardTransitionRef.current = "open"; setConfirmDiscard(true); };
  const keepEditing = () => { discardTransitionRef.current = "keep"; setConfirmDiscard(false); };
  const discardEmail = () => { discardTransitionRef.current = "discard"; setChannelDeviceId(null); setChannelAddressId(null); setCreateAddressId(null); setDestination(""); setLabels(["public"]); setReviewChannel(false); setConfirmDiscard(false); setChannelSaveError(""); };
  const emailValid = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(destination.trim());
  const deviceToRemove = devices.data?.devices.find((device) => device.id === removeId);
  const emailDevice = devices.data?.devices.find((device) => device.id === channelDeviceId);
  const savedEmail = (deviceId: string) => channelAddresses.data?.addresses.find((address) => address.deviceId === deviceId && address.channel === "email");
  const currentChannel = channelAddressId ? channelAddresses.data?.addresses.find((address) => address.id === channelAddressId) : undefined;
  // Unknown sensitivity keys are preserved in `labels` while the owner edits
  // known labels. Unsupported channel keys never get an editor entry point.
  const labelsEditable = true;
  const addedLabels = labels.filter((label) => !currentChannel?.approvedLabels.includes(label));
  const removedLabels = currentChannel?.approvedLabels.filter((label) => !labels.includes(label)) ?? [];
  const isWebPushChannel = editingChannel === "web_push";
  const safeDestination = (address: string) => isWebPushChannel ? t(strings.reach.copy.browserDestinationUnchanged) : address;
  const labelOnlyDiscardDraft = Boolean(currentChannel && destination.trim() === currentChannel.address && addedLabels.length > 0 && removedLabels.length === 0);
  const showDiscardDestinations = isWebPushChannel || !currentChannel || destination.trim() !== currentChannel.address;

  const saveChannel = async () => {
    const savedAddress = isWebPushChannel ? currentChannel?.address : destination.trim();
    if (!emailDevice || !savedAddress || (!isWebPushChannel && !emailValid) || labels.length === 0 || !labelsEditable) return;
    setChannelSaveError("");
    setChannelSaveMessage("");
    setSavingChannel(true);
    try {
      const stableId = channelAddressId ? null : (createAddressId ?? newStableAddressId());
      if (stableId) setCreateAddressId(stableId);
      const payload = { address: savedAddress, approvedLabels: labels };
      const id = channelAddressId ?? stableId!;
      const matchesSaved = (row: { id: string; deviceId: string; channel: string; address: string; approvedLabels: string[] } | undefined) => row?.id === id && row.deviceId === emailDevice.id && row.channel === editingChannel && row.address === payload.address && JSON.stringify(row.approvedLabels) === JSON.stringify(payload.approvedLabels);
      let responseUnknown = false;
      try {
        if (channelAddressId) await recipientsClient.updateChannelAddress({ id: channelAddressId, ...payload });
        else await recipientsClient.createChannelAddress({ id, deviceId: emailDevice.id, channel: editingChannel, ...payload });
      } catch { responseUnknown = true; }
      await queryClient.cancelQueries({ queryKey: ["recipient-channel-addresses"], exact: true });
      const latest = await queryClient.fetchQuery({ queryKey: ["recipient-channel-addresses"], queryFn: () => recipientsClient.listChannelAddresses({}), staleTime: 0 });
      if (!matchesSaved(latest.addresses.find((row) => row.id === id))) {
        const conflicting = !channelAddressId && latest.addresses.find((row) => row.deviceId === emailDevice.id && row.channel === editingChannel);
        if (conflicting) {
          setChannelSaveError(t(strings.reach.copy.channelConflict));
          await devices.refetch().catch(() => undefined);
          return;
        }
        setChannelSaveError(responseUnknown
          ? t(strings.reach.copy.channelSaveUncertain)
          : t(strings.reach.copy.channelSaveUnconfirmed));
        return;
      }
      setChannelSaveMessage(isWebPushChannel
        ? t(responseUnknown ? strings.reach.copy.webPushSaveConfirmedUncertain : strings.reach.copy.webPushSaveConfirmed)
        : t(responseUnknown ? strings.reach.copy.channelSaveConfirmedUncertain : strings.reach.copy.channelSaveConfirmed));
      setChannelDeviceId(null);
      setChannelAddressId(null);
      setCreateAddressId(null);
      setReviewChannel(false);
      await devices.refetch();
    } catch {
      setChannelSaveError(t(isWebPushChannel ? strings.reach.copy.webPushReadUnknown : strings.reach.copy.channelReadUnknown));
    } finally {
      setSavingChannel(false);
    }
  };
  const selectedDevice = devices.data?.devices.find((device) => device.id === selectedDeviceId) ?? devices.data?.devices[0];
  const showFocusedEntry = focusedEntry && devices.data?.devices.length === 0 && !preflight &&
    !pushMessage && !channelSaveMessage && !channelSaveError && !removeId && !emailDevice;
  const showPermissionRecovery = focusedEntry && permissionDenied && !permissionRecoveryDismissed;
  const showDeviceReadFailure = focusedEntry && devices.isError && !devices.data && !preflight &&
    !pushMessage && !channelSaveMessage && !channelSaveError && !removeId && !emailDevice &&
    !confirmDiscard && !removalResult;
  const showChannelReadGap = focusedEntry && channelReadGapFocused && Boolean(selectedDevice) &&
    channelAddresses.isError && !confirmDiscard && !deviceToRemove && !removalResult;
  const closeChannelReadGap = useCallback(() => {
    setChannelReadGapFocused(false);
  }, []);

  useEffect(() => {
    const discardTransition = discardTransitionRef.current;
    if (discardTransition === "open" && confirmDiscard) {
      discardReviewHeadingRef.current?.focus();
      discardTransitionRef.current = null;
      return;
    }
    if (discardTransition === "keep" && !confirmDiscard) {
      discardTriggerRef.current?.focus();
      discardTransitionRef.current = null;
      return;
    }
    if (discardTransition === "discard" && !confirmDiscard && !emailDevice) {
      (channelEntryTriggerIdRef.current ? document.getElementById(channelEntryTriggerIdRef.current) : null)?.focus();
      channelEntryTriggerIdRef.current = null;
      if (document.activeElement === document.body) document.getElementById("reach-heading")?.focus();
      discardTransitionRef.current = null;
      return;
    }
  }, [confirmDiscard, emailDevice]);

  useEffect(() => {
    if (focusedEntry && deviceToRemove) {
      restoreRemoveFocusRef.current = true;
      removeReviewHeadingRef.current?.focus();
      return;
    }
    if (!removeId && restoreRemoveFocusRef.current) {
      restoreRemoveFocusRef.current = false;
      const focusTarget = removalConfirmedRef.current
        ? document.getElementById("reach-heading")
        : removeTriggerIdRef.current ? document.getElementById(removeTriggerIdRef.current) : null;
      removeTriggerIdRef.current = null;
      removalConfirmedRef.current = false;
      focusTarget?.focus();
    }
  }, [deviceToRemove, focusedEntry, removeId]);

  useEffect(() => {
    if (!setContextHeader) return;
    if (confirmDiscard && emailDevice) {
      setContextHeader({
        title: emailDevice.name,
        backLabel: t(strings.reach.copy.contentAccess),
        onBack: keepEditing,
      });
      return () => setContextHeader(null);
    }
    if (showChannelReadGap && selectedDevice) {
      setContextHeader({
        title: selectedDevice.name,
        backLabel: t(strings.reach.copy.device),
        onBack: closeChannelReadGap,
      });
      return () => setContextHeader(null);
    }
    if (focusedEntry && removalResult) {
      setContextHeader({
        title: removalResult.device.name,
        backLabel: t(strings.reach.copy.devices),
        onBack: () => setRemovalResult(null),
      });
      return () => setContextHeader(null);
    }
    if (!focusedEntry || !deviceToRemove) {
      setContextHeader(null);
      return;
    }
    setContextHeader({
      title: deviceToRemove.name,
      backLabel: t(strings.reach.copy.devices),
      onBack: () => setRemoveId(null),
    });
    return () => setContextHeader(null);
  }, [closeChannelReadGap, confirmDiscard, deviceToRemove?.id, deviceToRemove?.name, emailDevice?.id, emailDevice?.name, focusedEntry, removalResult?.device.id, removalResult?.device.name, selectedDevice?.id, selectedDevice?.name, setContextHeader, showChannelReadGap, t]);

  useEffect(() => {
    if (!devices.data?.devices.length) { setSelectedDeviceId(""); return; }
    if (!devices.data.devices.some((device) => device.id === selectedDeviceId)) setSelectedDeviceId(devices.data.devices[0]?.id ?? "");
  }, [devices.data?.devices, selectedDeviceId]);

  useEffect(() => {
    if (channelReadGapFocused || !channelReadGapTriggerIdRef.current) return;
    document.getElementById(channelReadGapTriggerIdRef.current)?.focus();
    channelReadGapTriggerIdRef.current = null;
  }, [channelReadGapFocused, selectedDeviceId]);

  useEffect(() => {
    if (status.data?.permission === "denied") {
      setPermissionDenied(true);
    } else if (status.data?.permission) {
      setPermissionDenied(false);
      setPermissionRecoveryDismissed(false);
    }
  }, [status.data?.permission]);

  return (
    <section className={`flex flex-col gap-3 ${focusedEntry ? "pt-0" : ""}`} aria-labelledby={confirmDiscard ? "discard-channel-title" : "reach-heading"} data-testid="reach-settings">
      {confirmDiscard && emailDevice ? (
        <section className="flex flex-col gap-4" data-testid="reach-channel-discard-review">
          <div className="flex flex-col gap-2">
            <p className="text-sm text-app-muted-foreground">{t(strings.reach.copy.discardChannelContext, { name: emailDevice.name })}</p>
            <h2 id="discard-channel-title" ref={discardReviewHeadingRef} tabIndex={-1} className="text-2xl font-semibold">{t(strings.reach.copy.discardEmailTitle, { name: emailDevice.name })}</h2>
            <p className="text-sm">{labelOnlyDiscardDraft ? t(strings.reach.copy.discardAddsOnlyExplanation, { labels: addedLabels.map((label) => sensitivityLabel(label, t)).join(", ") }) : t(strings.reach.copy.discardExplanation)}</p>
          </div>
          <dl className="grid grid-cols-[minmax(7rem,0.8fr)_minmax(0,1.2fr)] gap-x-3 gap-y-3 border-y border-app-border py-3 text-sm">
            <dt className="text-app-muted-foreground">{t(strings.reach.copy.deviceId)}</dt><dd className="break-all text-end"><bdi dir="ltr">{emailDevice.id}</bdi></dd>
            {showDiscardDestinations && <><dt className="text-app-muted-foreground">{t(strings.reach.copy.savedDestination)}</dt><dd className="break-all text-end"><bdi dir="ltr">{safeDestination(currentChannel?.address ?? (channelAddressId ? t(strings.reach.copy.unknown) : t(strings.reach.copy.notSavedYet)))}</bdi></dd></>}
            <dt className="text-app-muted-foreground">{t(strings.reach.copy.savedFullContent)}</dt><dd className="text-end">{currentChannel ? (currentChannel.approvedLabels.length ? currentChannel.approvedLabels.map((label) => sensitivityLabel(label, t)).join(", ") : t(strings.reach.copy.unknown)) : channelAddressId ? t(strings.reach.copy.unknown) : t(strings.reach.copy.none)}</dd>
            {showDiscardDestinations && <><dt className="text-app-muted-foreground">{t(strings.reach.copy.draftDestination)}</dt><dd className="break-all text-end"><bdi dir="ltr">{safeDestination(destination.trim() || t(strings.reach.copy.noDestination))}</bdi></dd></>}
          </dl>
          <div className="flex flex-col gap-2 text-sm" data-testid="reach-discard-approval-diff">
            {channelAddressId && !currentChannel ? <p role="status">{t(strings.reach.copy.discardDiffUnknown)}</p> : <>
              {(addedLabels.length > 0 || removedLabels.length === 0) && <p><span className="font-medium">{t(strings.reach.copy.draftAdds)}</span> {addedLabels.length ? addedLabels.map((label) => sensitivityLabel(label, t)).join(", ") : t(strings.reach.copy.none)}</p>}
              {(removedLabels.length > 0 || addedLabels.length === 0) && <p><span className="font-medium">{t(strings.reach.copy.draftRemoves)}</span> {removedLabels.length ? removedLabels.map((label) => sensitivityLabel(label, t)).join(", ") : t(strings.reach.copy.none)}</p>}
            </>}
          </div>
          <p className="text-sm">{t(strings.reach.copy.discardReturnToDevices)}</p>
          <div className="grid grid-cols-2 gap-3 border-t border-app-border pt-3">
            <Button ref={discardTriggerRef} type="button" variant="primary" onClick={keepEditing}>{t(strings.reach.copy.keepEditing)}</Button>
            <Button type="button" variant="secondary" className="border-app-danger text-app-danger" onClick={discardEmail}>{t(strings.reach.copy.discardChanges)}</Button>
          </div>
        </section>
      ) : showChannelReadGap && selectedDevice ? (
        <>
          <div className="flex flex-col gap-5 sm:max-w-3xl" data-testid="reach-channel-read-gap">
            <p className="text-sm text-app-muted-foreground"><bdi dir="auto">{selectedDevice.name}</bdi> · <bdi dir="ltr">{selectedDevice.id}</bdi></p>
            <h1 id="reach-heading" className="-mt-3 text-2xl leading-8 font-bold tracking-tight">{t(strings.reach.copy.channelReadGapHeading)}</h1>
            <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-3 text-sm">
              <dt className="text-app-muted-foreground">{t(strings.reach.copy.deviceId)}</dt><dd className="break-all text-end"><bdi dir="ltr">{selectedDevice.id}</bdi></dd>
              <dt className="text-app-muted-foreground">{t(strings.reach.copy.machine)}</dt><dd className="break-all text-end"><bdi dir="ltr">{selectedDevice.machineId || t(strings.reach.copy.noMachine)}</bdi></dd>
              <dt className="text-app-muted-foreground">{t(selectedDevice.channels.length === 1 ? strings.reach.copy.channel : strings.reach.copy.channels)}</dt><dd className="text-end">{selectedDevice.channels.map((channel) => channelLabel(channel, t)).join(", ") || t(strings.reach.copy.none)}</dd>
            </dl>
            <div className="ms-4 border-s-4 py-2 ps-1" style={{ borderInlineStartColor: "var(--reach-guidance-rail)" }}>
              <p className="font-semibold">{t(strings.reach.copy.channelReadGapPreservedTitle)}</p>
              <p className="mt-2 text-sm leading-relaxed">{t(strings.reach.copy.channelReadGapPreservedBody)}</p>
            </div>
            <p className="text-sm leading-relaxed">{t(strings.reach.copy.channelReadGapEditHold)}</p>
          </div>
          <footer className="-mx-4 mt-10 grid grid-cols-1 gap-3 border-t border-app-border bg-app-surface px-4 py-3 sm:-mx-6 sm:grid-cols-2 sm:px-6">
            <div className="flex flex-col">
              <Button type="button" variant="primary" className="min-h-11 rounded-control" disabled aria-describedby="reach-channel-read-gap-action-description">{t(strings.reach.copy.editContent)}</Button>
              <span id="reach-channel-read-gap-action-description" className="sr-only">{t(strings.reach.copy.channelReadGap)}</span>
            </div>
            <Button type="button" variant="secondary" className="min-h-11 rounded-control border border-app-border" onClick={closeChannelReadGap}>{t(strings.reach.copy.backToDevice)}</Button>
          </footer>
        </>
      ) : showPermissionRecovery && !removalResult ? (
        <>
          <div className="flex flex-col gap-5 sm:max-w-3xl" data-testid="reach-permission-recovery">
            <p className="text-sm text-app-muted-foreground">{t(strings.reach.copy.thisBrowser)} · {t(strings.reach.copy.permissionDeniedValue)}</p>
            <h1 id="reach-heading" className="text-[1.75rem] leading-9 font-bold tracking-tight">{t(strings.reach.copy.permissionDeniedHeading)}</h1>
            <p className="text-sm">{t(strings.reach.copy.permissionDeniedBody)}</p>
            <aside role="status" className="border-s-4 py-2 ps-4" style={{ borderInlineStartColor: "var(--reach-guidance-rail)" }}>
              <p className="font-semibold">{t(strings.reach.copy.permissionDeniedReviewTitle)}</p>
              <p className="mt-2 text-sm leading-relaxed">{t(strings.reach.copy.permissionDeniedReviewBody)}</p>
            </aside>
            <p>{t(strings.reach.copy.permissionDeniedAvailable)}</p>
          </div>
          <footer className="-mx-4 mt-10 border-t border-app-border bg-app-surface px-4 py-3 sm:-mx-6 sm:px-6">
            <button data-testid="reach-permission-recovery-back" type="button" onClick={() => setPermissionRecoveryDismissed(true)} className="min-h-11 w-full rounded-lg border border-app-border px-4 text-center font-medium hover:bg-app-surface-raised">{t(strings.reach.copy.backToDevices)}</button>
          </footer>
        </>
      ) : showDeviceReadFailure ? (
        <>
          <div className="flex flex-col gap-4 sm:max-w-3xl" data-testid="reach-device-read-failure">
            <p className="text-sm text-app-muted-foreground">{t(strings.reach.copy.deviceListHeading)}</p>
            <h1 id="reach-heading" className="text-[1.75rem] leading-9 font-bold tracking-tight">{t(strings.reach.copy.deviceReadFailureTitle)}</h1>
            <aside role="alert" className="border-s-4 py-2 ps-4" style={{ borderInlineStartColor: "var(--reach-guidance-rail)" }}>
              <p className="font-semibold">{t(strings.reach.copy.deviceReadError)}</p>
              <p className="mt-2 text-sm leading-relaxed">{t(strings.reach.copy.deviceReadFailureBody)}</p>
            </aside>
            <p className="text-sm leading-relaxed">{t(strings.reach.copy.deviceReadFailureHelp)}</p>
          </div>
          <footer className="-mx-4 mt-6 grid grid-cols-2 gap-3 border-t border-app-border bg-app-surface px-4 py-3 sm:-mx-6 sm:px-6">
            <Button type="button" variant="primary" className="min-h-11 rounded-control" disabled={devices.isFetching} onClick={() => void devices.refetch()}>{t(strings.reach.copy.retry)}</Button>
            <Link to={preferencesHref} className="inline-flex min-h-11 items-center justify-center rounded-lg border border-app-border px-4 text-center font-medium hover:bg-app-surface-raised">{t(strings.reach.preferences)}</Link>
          </footer>
        </>
      ) : showFocusedEntry ? (
        <>
          <div className="flex flex-col gap-3.5 sm:max-w-3xl">
            <p className="-mb-2 text-sm text-app-muted-foreground">{t(strings.reach.emptyKicker)}</p>
            <div className="-mb-0.5 flex flex-col gap-2.5">
              <h1 id="reach-heading" className="text-[1.75rem] leading-9 font-bold tracking-tight">{t(strings.reach.emptyHeading)}</h1>
              <p data-testid="reach-empty-devices" className="text-base leading-relaxed">{t(strings.reach.emptyAddBrowser)}</p>
              <div role="status" className="border-s-4 py-2 ps-4" style={{ borderInlineStartColor: "var(--reach-guidance-rail)" }}>
                <p className="font-semibold">{t(strings.reach.copy.deliberateTitle)}</p>
                <p className="mt-2 text-sm leading-relaxed">{t(strings.reach.emptyScopeBody)}</p>
              </div>
              <Link to={preferencesHref} className="self-start text-sm font-medium text-app-primary underline underline-offset-4">{t(strings.reach.preferences)}</Link>
            </div>
          </div>
          <footer className="-mx-4 mt-6 grid grid-cols-2 gap-3 border-t border-app-border bg-app-surface px-4 py-3 sm:-mx-6 sm:px-6">
            {signedIn && status.data?.supported && status.data.permission !== "denied" && !permissionDenied ? (
              <Button data-testid={selectors.reach.addBrowser} className="min-h-11 rounded-control" type="button" variant="primary" disabled={requestingPermission} onClick={() => setPreflight(true)}>{t(strings.reach.addThisBrowser)}</Button>
            ) : <span />}
            <Link to="/" className="inline-flex min-h-11 items-center justify-center rounded-lg border border-app-border px-4 text-center font-medium hover:bg-app-surface-raised">{t(strings.reach.backToRequests)}</Link>
          </footer>
        </>
      ) : removalResult && focusedEntry ? <>
        <div className="flex flex-col gap-6 sm:max-w-3xl" data-testid="reach-device-removed" role="group" aria-labelledby="reach-heading">
          <div className="flex flex-col gap-2">
            <p className="text-sm text-app-muted-foreground"><bdi dir="auto">{removalResult.device.name}</bdi> · <bdi dir="ltr">{removalResult.device.id}</bdi></p>
            <h1 id="reach-heading" tabIndex={-1} className="text-2xl leading-8 font-bold tracking-tight sm:text-[1.75rem] sm:leading-9">{t(strings.reach.copy.deviceRemovedTitle)}</h1>
          </div>
          <aside role="status" className="border-s-4 py-2 ps-4" style={{ borderInlineStartColor: "var(--reach-success-rail)" }}>
            <p className="font-semibold">{t(strings.reach.copy.deviceRemovalConfirmed)}</p>
            <p className="mt-2 text-sm leading-relaxed">{t(strings.reach.copy.deviceRemovalConfirmedBody, { name: removalResult.device.name })}</p>
          </aside>
          <p className="text-sm leading-relaxed">{t(strings.reach.copy.deviceRemovalPreservesPermission)}</p>
          {removalResult.responseUnknown && <p role="status" className="text-sm text-app-muted-foreground">{t(strings.reach.copy.deviceRemoveUncertain)}</p>}
          {removalResult.remaining.length > 0 ? <section className="flex flex-col gap-3" aria-labelledby="remaining-device-heading">
            <h2 id="remaining-device-heading" className="text-xl font-semibold">{t(strings.reach.copy.remainingDevices)}</h2>
            <ul className="flex flex-col gap-2">{removalResult.remaining.map((device) => <li key={device.id}>
              <button type="button" onClick={() => { setSelectedDeviceId(device.id); setRemovalResult(null); }} className="flex min-h-14 w-full items-center justify-between gap-3 border-b border-app-border py-3 text-start">
                <span className="flex min-w-0 flex-col"><bdi dir="auto" className="font-semibold">{device.name}</bdi><span className="text-sm text-app-muted-foreground"><bdi dir="ltr">{device.id}</bdi>{device.channels.length > 0 ? ` · ${device.channels.map((channel) => channelLabel(channel, t)).join(", ")}` : ""}</span></span>
                <span aria-hidden="true">›</span>
              </button>
            </li>)}</ul>
          </section> : <p className="text-sm text-app-muted-foreground">{t(strings.reach.copy.noRemainingDevices)}</p>}
          <footer className="-mx-4 mt-4 border-t border-app-border bg-app-surface px-4 py-3 sm:-mx-6 sm:px-6">
            <Button data-testid="reach-device-removed-back" type="button" variant="primary" className="min-h-11 w-full rounded-control" onClick={() => setRemovalResult(null)}>{t(strings.reach.copy.backToDevices)}</Button>
          </footer>
        </div>
      </> : deviceToRemove && focusedEntry ? <>
        <div className="flex flex-col gap-6 sm:max-w-3xl" data-testid="reach-remove-review" role="group" aria-labelledby="reach-heading">
          <div className="flex flex-col gap-2">
            <p className="text-sm text-app-muted-foreground"><bdi dir="auto">{deviceToRemove.name}</bdi> · <bdi dir="ltr">{deviceToRemove.id}</bdi></p>
            <h1 ref={removeReviewHeadingRef} tabIndex={-1} id="reach-heading" className="text-[1.75rem] leading-9 font-bold tracking-tight">{t(strings.reach.copy.removeTitle, { name: deviceToRemove.name })}</h1>
            <p className="text-sm leading-relaxed">{t(strings.reach.copy.removeEffect)}</p>
          </div>
          <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-3 text-sm">
            <dt className="text-app-muted-foreground">{t(strings.reach.copy.deviceId)}</dt><dd className="break-all text-end"><bdi dir="ltr">{deviceToRemove.id}</bdi></dd>
            <dt className="text-app-muted-foreground">{t(strings.reach.copy.machine)}</dt><dd className="break-all text-end"><bdi dir="auto">{deviceToRemove.machineId || t(strings.reach.copy.noMachine)}</bdi></dd>
            <dt className="text-app-muted-foreground">{t(strings.reach.copy.channel)}</dt><dd className="text-end">{deviceToRemove.channels.length ? deviceToRemove.channels.map((channel) => channelLabel(channel, t)).join(", ") : t(strings.reach.copy.none)}</dd>
          </dl>
          <aside className="border-s-4 py-2 ps-4" style={{ borderInlineStartColor: "var(--reach-guidance-rail)" }}>
            <p className="font-semibold">{t(strings.reach.copy.removePermissionTitle)}</p>
            <p className="mt-2 text-sm leading-relaxed">{t(strings.reach.copy.removePermissionBody)}</p>
          </aside>
        <footer className="-mx-4 mt-10 grid grid-cols-2 gap-3 border-t border-app-border bg-app-surface px-4 py-3 sm:-mx-6 sm:px-6">
          <Button type="button" variant="secondary" className="min-h-11 rounded-control border border-app-danger text-app-danger" disabled={remove.isPending} onClick={() => remove.mutate({ id: deviceToRemove.id, device: deviceToRemove })}>{remove.isPending ? t(strings.reach.copy.removing) : t(strings.reach.copy.removeDevice)}</Button>
          <Button type="button" variant="secondary" className="min-h-11 rounded-control border border-app-border" disabled={remove.isPending} onClick={() => setRemoveId(null)}>{t(strings.reach.copy.keepDevice)}</Button>
        </footer>
        {pushMessage && <p role="status" aria-live="polite" className="text-sm text-app-muted-foreground">{pushMessage}</p>}
        </div>
      </>
      : <>{emailDevice && channelAddressId ? <div className="flex flex-col gap-2 sm:max-w-3xl" data-testid="reach-email-focused-header">
        <p className="text-sm text-app-muted-foreground">{t(isWebPushChannel ? strings.reach.copy.reviewWebPushLabel : strings.reach.copy.reviewEmailLabel)} · <bdi dir="ltr">{emailDevice.name}</bdi></p>
        <h1 id="reach-heading" className="text-[1.75rem] leading-9 font-bold tracking-tight">{t(isWebPushChannel ? strings.reach.copy.contentAccess : strings.reach.copy.checkDestination)}</h1>
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm"><dt className="text-app-muted-foreground">{t(strings.reach.copy.machine)}</dt><dd className="text-end">{emailDevice.machineId || t(strings.reach.copy.noMachine)}</dd><dt className="text-app-muted-foreground">{t(strings.reach.copy.channel)}</dt><dd className="text-end">{channelLabel(editingChannel, t)}</dd>{isWebPushChannel && <><dt className="text-app-muted-foreground">{t(strings.reach.copy.destination)}</dt><dd className="text-end">{t(strings.reach.copy.browserDestinationUnchanged)}</dd></>}</dl>
      </div> : <>
      <p className="text-sm font-semibold uppercase text-app-muted-foreground">{t(strings.reach.title)}</p>
      <h3 id="reach-heading" tabIndex={-1} className="text-2xl font-semibold">{t(strings.reach.heading)}</h3>
      <p className="text-sm text-app-muted-foreground">{t(strings.reach.copy.overview)}</p>
      {devices.data?.devices.length === 0 && <aside className="border-s-4 border-app-warning ps-4 py-3"><p className="font-semibold">{t(strings.reach.copy.deliberateTitle)}</p><p className="text-sm">{t(strings.reach.copy.deliberateBody)}</p></aside>}
      <SessionPanel />
      {!signedIn && <p className="text-sm text-app-muted-foreground">{t(strings.reach.copy.signInManage)}</p>}
      {signedIn && status.data?.supported && status.data.permission !== "denied" && !permissionDenied && (
        <Button data-testid={selectors.reach.addBrowser} className="min-h-11 rounded-control" type="button" variant="primary" disabled={requestingPermission} onClick={() => setPreflight(true)}>{t(strings.reach.addThisBrowser)}</Button>
      )}
      {status.data && !status.data.supported && <p role="status" className="rounded border border-app-border p-3 text-sm">{t(strings.reach.copy.browserUnsupported)}</p>}
      {status.data?.permission === "denied" && <p role="status" className="rounded border border-app-border p-3 text-sm">{t(strings.reach.copy.permissionDenied)}</p>}
      {preflight && (
        <div role="group" aria-labelledby="enroll-review-title" className="flex flex-col gap-3 rounded-lg border border-app-border bg-app-surface-raised p-4" data-testid="reach-enroll-preflight">
          <h4 id="enroll-review-title" className="font-semibold">{t(strings.reach.copy.reviewTitle)}</h4>
          <p className="text-sm">{t(strings.reach.copy.scopeDisclosure)}</p>
          <p className="text-sm text-app-muted-foreground">{t(strings.reach.copy.permissionBoundary)}</p>
          <div className="flex flex-wrap gap-2"><Button type="button" variant="primary" disabled={!status.data?.supported || status.data.permission === "denied"} onClick={() => void enableBrowser()}>{t(strings.reach.copy.continuePrompt)}</Button><Button type="button" variant="secondary" onClick={() => setPreflight(false)}>{t(strings.reach.copy.cancel)}</Button></div>
        </div>
      )}
      {enrolling && <p role="status" aria-live="polite" className="text-sm">{t(strings.reach.copy.registering)}</p>}
      {requestingPermission && <p role="status" aria-live="polite" className="text-sm">{t(strings.reach.copy.waitingPermission)}</p>}
      {pushMessage && <p role="status" aria-live="polite" className="text-sm text-app-muted-foreground">{pushMessage}</p>}
      {channelSaveMessage && <p role="status" aria-live="polite" className="text-sm text-app-muted-foreground">{channelSaveMessage}</p>}
      <ConnectionStatusPanel status={status.data} loading={status.isLoading} error={status.isError} refreshing={status.isFetching} onRefresh={() => void status.refetch()} onUpdate={() => update.mutate()} updating={update.isPending} />
      <div className="flex items-center justify-between gap-3">
        <h4 className="font-semibold">{t(strings.reach.copy.deviceListHeading)}</h4>
        <button type="button" data-testid={selectors.reach.refreshDevices} className="inline-flex items-center gap-2 rounded-lg border border-app-border px-3 py-1.5 text-sm" onClick={() => void devices.refetch()} aria-label={t(strings.reach.copy.refreshDevices)}><RefreshCw className={`h-4 w-4 ${devices.isFetching ? "animate-spin" : ""}`} aria-hidden="true" />{t(strings.reach.copy.refresh)}</button>
      </div>
      {devices.isLoading && <p role="status" className="text-sm">{t(strings.reach.copy.loadingDevices)}</p>}
      {devices.isError && !devices.data && <div role="alert" className="rounded border border-app-border p-3"><p>{t(strings.reach.copy.deviceReadError)}</p><Button type="button" variant="secondary" onClick={() => void devices.refetch()}>{t(strings.reach.copy.retry)}</Button></div>}
      {devices.isError && devices.data && <p role="status" className="rounded border border-app-warning p-3 text-sm">{t(strings.reach.copy.staleDevices)}</p>}
      {devices.data && devices.data.devices.length === 0 && <p data-testid="reach-empty-devices" className="rounded border border-app-border p-4 text-sm">{t(strings.reach.copy.noDevices)}</p>}
      {devices.data && devices.data.devices.length > 0 && selectedDevice && <div className="grid gap-4 lg:grid-cols-[minmax(15rem,0.8fr)_minmax(0,1.6fr)]">
        <ul aria-label={t(strings.reach.copy.deviceListLabel)} className="flex flex-col gap-2">
          {devices.data.devices.map((device) => <li key={device.id}><button id={`reach-device-option-${device.id}`} type="button" aria-pressed={selectedDevice.id === device.id} onClick={(event) => { setSelectedDeviceId(device.id); if (focusedEntry && channelAddresses.isError) { channelReadGapTriggerIdRef.current = event.currentTarget.id; setChannelReadGapFocused(true); } }} className={`w-full rounded-lg border p-3 text-start ${selectedDevice.id === device.id ? "border-app-primary bg-app-surface-raised" : "border-app-border"}`}><span className="block font-semibold"><bdi dir="auto">{device.name}</bdi></span><span className="block text-xs text-app-muted-foreground">{device.channels.length ? device.channels.map((channel) => channelLabel(channel, t)).join(", ") : t(strings.reach.copy.noChannels)}</span><span className="block text-xs text-app-muted-foreground">{device.machineId ? <>{t(strings.reach.copy.machine)} <bdi dir="ltr">{device.machineId}</bdi></> : t(strings.reach.copy.noMachine)}</span></button></li>)}
        </ul>
        <article className="flex flex-col gap-3 rounded-lg border border-app-border bg-app-surface-raised p-4" data-testid={`reach-device-${selectedDevice.id}`}>
          <div><h5 className="font-semibold">{selectedDevice.name}</h5><p className="text-xs text-app-muted-foreground">{t(strings.reach.copy.deviceId)} <bdi dir="ltr">{selectedDevice.id}</bdi></p><p className="text-xs text-app-muted-foreground">{selectedDevice.machineId ? <>{t(strings.reach.copy.machine)} <bdi dir="ltr">{selectedDevice.machineId}</bdi></> : t(strings.reach.copy.noMachine)}</p></div>
          <div><p className="text-sm font-medium">{t(strings.reach.copy.channels)}</p>{channelAddresses.isLoading && <p role="status" className="text-sm">{t(strings.reach.copy.channelReadLoading)}</p>}{channelAddresses.isError && <p role="alert" className="text-sm">{t(strings.reach.copy.channelReadError)} {channelAddresses.data ? t(strings.reach.copy.staleChannelList) : t(strings.reach.copy.channelReadGap)}</p>}{selectedDevice.channels.length ? <ul className="mt-2 flex flex-col gap-2">{selectedDevice.channels.map((channel) => { const address = channelAddresses.data?.addresses.find((row) => row.deviceId === selectedDevice.id && row.channel === channel); return <li key={channel} className="rounded border border-app-border p-3"><span className="font-medium">{channelLabel(channel, t)}</span>{address ? <><p className="mt-1 break-all text-xs text-app-muted-foreground">{t(strings.reach.copy.savedDestination)} <bdi dir="ltr">{channel === "web_push" ? t(strings.reach.copy.browserEndpointPrivate) : address.address}</bdi></p><p className="text-xs text-app-muted-foreground">{t(strings.reach.copy.approvedLabels)} {address.approvedLabels.length ? address.approvedLabels.map((label) => sensitivityLabel(label, t)).join(", ") : t(strings.reach.copy.unknown)}</p>{(channel === "email" || channel === "web_push") && signedIn && <Button id={channel === "email" ? `reach-channel-entry-${selectedDevice.id}` : `reach-channel-entry-${selectedDevice.id}-${channel}`} className="min-h-11" type="button" variant="secondary" disabled={channelAddresses.isError || channelAddresses.isFetching} onClick={(event) => openExistingChannel(selectedDevice.id, address, event.currentTarget)}>{t(strings.reach.copy.editContent)}</Button>}</> : <><p className="mt-1 text-xs text-app-muted-foreground">{t(strings.reach.copy.channelUnknown)}</p><Button type="button" variant="secondary" className="min-h-11" disabled aria-describedby={`reach-read-gap-${selectedDevice.id}-${channel}`}>{t(strings.reach.copy.editContent)}</Button><span id={`reach-read-gap-${selectedDevice.id}-${channel}`} className="sr-only">{t(strings.reach.copy.channelReadGap)}</span></>}</li>; })}</ul> : <p className="text-sm text-app-muted-foreground">{t(strings.reach.copy.channelNamesMissing)}</p>}</div>
          <aside data-testid="reach-device-readiness-unknown" className="border-s-4 border-app-warning ps-4 py-3"><p className="text-sm font-medium">{t(strings.reach.copy.readiness)}</p><p className="text-sm text-app-muted-foreground">{t(strings.reach.copy.readinessUnknown)}</p></aside>
          {signedIn && <div className="flex flex-wrap gap-2"><Button id={`reach-email-entry-${selectedDevice.id}`} className="min-h-11" type="button" variant="secondary" disabled={devices.isError || devices.isFetching || channelAddresses.isError || channelAddresses.isFetching || Boolean(selectedDevice.channels.includes("email") && !savedEmail(selectedDevice.id))} onClick={(event) => { const email = savedEmail(selectedDevice.id); openEmail(selectedDevice.id, email ? { id: email.id, address: email.address, approvedLabels: email.approvedLabels } : undefined, event.currentTarget); }}>{selectedDevice.channels.includes("email") ? t(strings.reach.copy.editEmail) : t(strings.reach.copy.addEmail)}</Button><Button id={`reach-remove-trigger-${selectedDevice.id}`} className="min-h-11" type="button" variant="secondary" disabled={devices.isError || devices.isFetching} onClick={() => { removalConfirmedRef.current = false; removeTriggerIdRef.current = `reach-remove-trigger-${selectedDevice.id}`; setRemoveId(selectedDevice.id); }}>{t(strings.reach.copy.removeDevice)}</Button></div>}
        </article>
      </div>}
      </>}
      {deviceToRemove && !focusedEntry && <div role="group" aria-labelledby="remove-device-title" className="flex flex-col gap-3 rounded-lg border border-app-danger p-4" data-testid="reach-remove-review"><h4 id="remove-device-title" className="font-semibold">{t(strings.reach.copy.removeTitle, { name: deviceToRemove.name })}</h4><p className="text-sm">{t(strings.reach.copy.removeEffect)}</p><p className="text-sm">{t(strings.reach.copy.removePermissionBody)}</p><p className="text-xs">{t(strings.reach.copy.knownChannels, { channels: deviceToRemove.channels.join(", ") || t(strings.reach.copy.none)})}</p><div className="flex flex-wrap gap-2"><Button type="button" variant="primary" disabled={remove.isPending} onClick={() => remove.mutate({ id: deviceToRemove.id, device: deviceToRemove })}>{remove.isPending ? t(strings.reach.copy.removing) : t(strings.reach.copy.removeDevice)}</Button><Button type="button" variant="secondary" disabled={remove.isPending} onClick={() => setRemoveId(null)}>{t(strings.reach.copy.keepDevice)}</Button></div></div>}
      {emailDevice && <div role={confirmDiscard ? "region" : "group"} aria-labelledby={confirmDiscard ? "discard-channel-title" : "email-channel-title"} className={`flex flex-col gap-3 ${channelAddressId ? "sm:max-w-3xl" : "rounded-lg border border-app-border p-4"}`} data-testid="reach-channel-editor">
        {confirmDiscard ? <section className="flex flex-col gap-4" data-testid="reach-channel-discard-review">
          <div className="flex flex-col gap-2">
            <p className="text-sm text-app-muted-foreground">{t(strings.reach.copy.discardChannelContext, { name: emailDevice.name })}</p>
            <h2 id="discard-channel-title" ref={discardReviewHeadingRef} tabIndex={-1} className="text-2xl font-semibold">{t(strings.reach.copy.discardEmailTitle, { name: emailDevice.name })}</h2>
            <p className="text-sm">{t(strings.reach.copy.discardExplanation)}</p>
          </div>
          <dl className="grid grid-cols-[minmax(7rem,0.8fr)_minmax(0,1.2fr)] gap-x-3 gap-y-3 border-y border-app-border py-3 text-sm">
            <dt className="text-app-muted-foreground">{t(strings.reach.copy.deviceId)}</dt><dd className="break-all text-end"><bdi dir="ltr">{emailDevice.id}</bdi></dd>
            <dt className="text-app-muted-foreground">{t(strings.reach.copy.savedDestination)}</dt><dd className="break-all text-end"><bdi dir="ltr">{currentChannel?.address ?? (channelAddressId ? t(strings.reach.copy.unknown) : t(strings.reach.copy.notSavedYet))}</bdi></dd>
            <dt className="text-app-muted-foreground">{t(strings.reach.copy.savedFullContent)}</dt><dd className="text-end">{currentChannel ? (currentChannel.approvedLabels.length ? currentChannel.approvedLabels.map((label) => sensitivityLabel(label, t)).join(", ") : t(strings.reach.copy.unknown)) : channelAddressId ? t(strings.reach.copy.unknown) : t(strings.reach.copy.none)}</dd>
            <dt className="text-app-muted-foreground">{t(strings.reach.copy.draftDestination)}</dt><dd className="break-all text-end"><bdi dir="ltr">{destination.trim() || t(strings.reach.copy.noDestination)}</bdi></dd>
          </dl>
          <div className="flex flex-col gap-2 text-sm" data-testid="reach-discard-approval-diff">
            {channelAddressId && !currentChannel ? <p role="status">{t(strings.reach.copy.discardDiffUnknown)}</p> : <>
              <p><span className="font-medium">{t(strings.reach.copy.draftAdds)}</span> {addedLabels.length ? addedLabels.map((label) => sensitivityLabel(label, t)).join(", ") : t(strings.reach.copy.none)}</p>
              <p><span className="font-medium">{t(strings.reach.copy.draftRemoves)}</span> {removedLabels.length ? removedLabels.map((label) => sensitivityLabel(label, t)).join(", ") : t(strings.reach.copy.none)}</p>
            </>}
          </div>
          <div className="grid grid-cols-2 gap-3 border-t border-app-border pt-3">
            <Button ref={discardTriggerRef} type="button" variant="primary" onClick={keepEditing}>{t(strings.reach.copy.keepEditing)}</Button>
            <Button type="button" variant="secondary" className="border-app-danger text-app-danger" onClick={discardEmail}>{t(strings.reach.copy.discardChanges)}</Button>
          </div>
        </section> : <>
        {channelAddressId ? <h2 id="email-channel-title" className="sr-only">{t(isWebPushChannel ? strings.reach.copy.webPushEditTitle : strings.reach.copy.emailEditTitle)}</h2> : <><h4 id="email-channel-title" className="font-semibold">{t(strings.reach.copy.emailAddTitle)}</h4><p className="text-xs text-app-muted-foreground">{t(strings.reach.copy.emailFor)} <bdi dir="ltr">{emailDevice.name}</bdi>. {t(strings.reach.copy.emailScope)}</p></>}
        {!reviewChannel ? <>
          {!isWebPushChannel && <FormField
            label={t(strings.reach.copy.emailAddress)}
            id="reach-email-destination"
            description={t(strings.reach.copy.emailAddressHelp)}
            error={destination.length > 0 && !emailValid ? t(strings.reach.copy.invalidEmail) : undefined}
            control={(
              <input
                type="email"
                dir="ltr"
                autoComplete="email"
                value={destination}
                onChange={(event) => setDestination(event.target.value)}
                aria-invalid={destination.length > 0 && !emailValid}
                className="min-h-11 w-full rounded-control border border-app-border bg-app-background px-3"
              />
            )}
          />}
          {!isWebPushChannel && destination.length > 0 && !emailValid && (
            <aside role="status" className="border-s-4 border-app-warning py-2 ps-4" data-testid="reach-invalid-email-preserves-saved">
              <p className="font-semibold">{t(strings.reach.copy.invalidEmailTitle)}</p>
              <p className="mt-1 text-sm">{t(currentChannel ? strings.reach.copy.invalidEmailExistingHelp : strings.reach.copy.invalidEmailNewHelp)}</p>
            </aside>
          )}
          <fieldset className="flex flex-col gap-1"><legend className={isWebPushChannel ? "text-xl font-bold tracking-tight" : "text-sm font-medium"}>{t(strings.reach.copy.fullContentPermitted)}</legend>{LABELS.map((label) => <label key={label} className={isWebPushChannel ? "flex min-h-11 items-center gap-3 text-base" : "flex min-h-11 items-center gap-2 text-sm"}><input type="checkbox" className={isWebPushChannel ? "h-5 w-5 shrink-0" : undefined} checked={labels.includes(label)} onChange={(event) => setLabels(event.target.checked ? [...labels, label] : labels.filter((item) => item !== label))} />{sensitivityLabel(label, t)}</label>)}</fieldset>
          {currentChannel && <p role="status" aria-live="polite" className="rounded border border-app-border p-3 text-sm" data-testid="reach-content-access-draft-diff">
            {t(removedLabels.length ? strings.reach.copy.savedDraftSummaryWithRemovals : strings.reach.copy.savedDraftSummary, {
              saved: currentChannel.approvedLabels.filter((label) => LABELS.includes(label as typeof LABELS[number])).map((label) => sensitivityLabel(label, t)).join(", ") || t(strings.reach.copy.none),
              added: addedLabels.map((label) => sensitivityLabel(label, t)).join(", ") || t(strings.reach.copy.none),
              removed: removedLabels.map((label) => sensitivityLabel(label, t)).join(", "),
            })}
          </p>}
          {currentChannel?.approvedLabels.some((label) => !LABELS.includes(label as typeof LABELS[number])) && <p role="status" className="rounded border border-app-warning p-3 text-sm" data-testid="reach-unknown-label-preserved">{t(strings.reach.copy.unknownApprovalPreserved, { labels: currentChannel.approvedLabels.filter((label) => !LABELS.includes(label as typeof LABELS[number])).join(", ") })}</p>}
          {!labels.length && <p role="status" className="rounded border border-app-warning p-3 text-sm">{t(strings.reach.copy.pointerOnlyHeld)}</p>}
          <div className="flex flex-wrap gap-2"><Button type="button" variant="primary" disabled={(!isWebPushChannel && !emailValid) || labels.length === 0} onClick={() => setReviewChannel(true)}>{t(channelAddressId ? strings.reach.copy.reviewChanges : strings.reach.copy.reviewContentAccess)}</Button><Button ref={discardTriggerRef} type="button" variant="secondary" onClick={requestDiscard}>{t(channelAddressId ? strings.reach.copy.cancel : strings.reach.copy.discard)}</Button></div>
        </> : <>
          {isWebPushChannel && addedLabels.length === 1 && (addedLabels[0] === "sensitive" || addedLabels[0] === "critical")
            ? <h2 className="text-base font-semibold">{t(strings.reach.copy.reviewWebPushSensitivityTitle, { label: sensitivityLabel(addedLabels[0], t) })}</h2>
            : <p className="text-sm">{isWebPushChannel ? t(strings.reach.copy.reviewWebPushExisting, { name: emailDevice.name }) : currentChannel ? t(strings.reach.copy.reviewEmailExisting) : t(strings.reach.copy.reviewEmailNew)} {!isWebPushChannel && <><bdi dir="ltr">{emailDevice.name}</bdi>.</>}</p>}
          {currentChannel && <p className="text-sm"><span className="font-medium">{t(isWebPushChannel ? strings.reach.copy.contentAccessBefore : strings.reach.copy.currentlyApproved)}</span> {currentChannel.approvedLabels.length ? (isWebPushChannel ? LABELS.filter((label) => currentChannel.approvedLabels.includes(label)) : currentChannel.approvedLabels).map((label) => sensitivityLabel(label, t)).join(", ") : t(strings.reach.copy.unknown)}</p>}
          {!isWebPushChannel && <p className="break-all text-sm"><span className="font-medium">{t(strings.reach.copy.destination)}</span> <bdi dir="ltr">{safeDestination(destination.trim())}</bdi></p>}
          <p className="text-sm"><span className="font-medium">{t(isWebPushChannel ? strings.reach.copy.contentAccessAfter : strings.reach.copy.fullContentAfterSave)}</span> {(isWebPushChannel ? LABELS.filter((label) => labels.includes(label)) : labels).map((label) => sensitivityLabel(label, t)).join(", ")}</p>
          {isWebPushChannel && addedLabels.some((label) => label === "sensitive" || label === "critical") && <div role="alert" className="rounded border border-app-warning p-3 text-sm"><p>{t(strings.reach.copy.moreContentOutsideHub)}</p><p>{t(strings.reach.copy.browserMayDisplaySensitive)}</p></div>}
          {isWebPushChannel && currentChannel?.approvedLabels.some((label) => !LABELS.includes(label as typeof LABELS[number])) && <p className="text-sm">{t(strings.reach.copy.reviewUnknownLabelsPreserved, { labels: currentChannel.approvedLabels.filter((label) => !LABELS.includes(label as typeof LABELS[number])).join(", ") })}</p>}
          {isWebPushChannel && !labels.includes("critical") && <p className="text-sm">{t(strings.reach.copy.criticalRemainsUnchecked)}</p>}
          {currentChannel && (addedLabels.length > 0 || removedLabels.length > 0) && !isWebPushChannel
            ? <p role="status" className="rounded border border-app-warning p-3 text-sm">{t(strings.reach.copy.approvalChange)} {addedLabels.length > 0 ? t(strings.reach.copy.adds, { labels: addedLabels.join(", ") }) : t(strings.reach.copy.addsNone)}{removedLabels.length > 0 ? `; ${t(strings.reach.copy.removes, { labels: removedLabels.join(", ") })}` : `; ${t(strings.reach.copy.removesNone)}`}. {t(strings.reach.copy.approvalSensitivity)}</p>
            : !isWebPushChannel && <p className="text-sm">{currentChannel ? t(strings.reach.copy.approvedUnchanged) : t(strings.reach.copy.approveNewDestination)}</p>}
          {!isWebPushChannel && <p className="text-xs text-app-muted-foreground">{t(strings.reach.copy.onlyTheseLabels)}</p>}
          {channelSaveError && <p role="alert" className="rounded border border-app-danger p-3 text-sm">{channelSaveError}</p>}
          <div className="flex flex-wrap gap-2"><Button type="button" variant="primary" disabled={savingChannel || !labelsEditable} onClick={() => void saveChannel()}>{savingChannel ? t(strings.reach.copy.saving) : t(isWebPushChannel ? strings.reach.copy.saveContentAccess : strings.reach.copy.saveDestination)}</Button><Button type="button" variant="secondary" disabled={savingChannel} onClick={() => setReviewChannel(false)}>{t(strings.reach.copy.keepEditing)}</Button><Button ref={discardTriggerRef} type="button" variant="secondary" disabled={savingChannel} onClick={requestDiscard}>{t(strings.reach.copy.discard)}</Button></div>
        </>}
        </>}
      </div>}
      {!confirmDiscard && !channelDeviceId && <p className="text-xs text-app-muted-foreground">{t(strings.reach.copy.readinessEvidence)}</p>}
      </>}
    </section>
  );
}

function ConnectionStatusPanel({ status, loading, error, refreshing, onRefresh, onUpdate, updating }: {
  status?: ConnectionStatus; loading: boolean; error: boolean; refreshing: boolean; onRefresh: () => void; onUpdate: () => void; updating: boolean;
}) {
  const { t } = useTranslation();
  return <div className="flex flex-col gap-2 rounded-lg border border-app-border bg-app-surface-raised p-3" data-testid="connection-status">
    <div className="flex items-center justify-between gap-2"><h4 className="font-semibold">{t(strings.reach.copy.thisBrowser)}</h4><button type="button" onClick={onRefresh} className="rounded border border-app-border px-3 py-1 text-sm">{refreshing ? t(strings.reach.copy.checking) : t(strings.reach.copy.checkAgain)}</button></div>
    {loading && <p role="status" className="text-sm">{t(strings.reach.copy.checkingStatus)}</p>}
    {error && <p role="alert" className="text-sm">{t(strings.reach.copy.registrationStatusUnavailable)}</p>}
    {status && <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm"><dt className="text-app-muted-foreground">{t(strings.reach.copy.browserSupport)}</dt><dd>{status.supported ? t(strings.reach.copy.available) : t(strings.reach.copy.unsupported)}</dd><dt className="text-app-muted-foreground">{t(strings.reach.copy.permission)}</dt><dd>{permissionLabel(status.permission, t)}</dd><dt className="text-app-muted-foreground">{t(strings.reach.copy.browserSubscription)}</dt><dd data-testid="connection-subscription-value">{status.subscribed ? <>{t(strings.reach.copy.present)} (<bdi dir="ltr">{status.endpointHost}</bdi>)</> : t(strings.reach.copy.notSubscribed)}</dd><dt className="text-app-muted-foreground">{t(strings.reach.copy.serviceWorker)}</dt><dd>{!status.controlled ? t(strings.reach.copy.noActiveWorker) : status.stale ? t(strings.reach.copy.staleWorker, { running: status.runningVersion ?? t(strings.reach.copy.unknown), available: status.availableVersion ?? t(strings.reach.copy.unknown) }) : t(strings.reach.copy.currentWorker, { version: status.runningVersion ?? status.availableVersion ?? t(strings.reach.copy.unknown) })}</dd></dl>}
    {status && !status.supported && <p className="text-xs text-app-muted-foreground">{t(strings.reach.copy.iphoneHomeScreen)}</p>}
    {status?.stale && <div className="flex flex-col gap-2"><Button type="button" variant="primary" disabled={updating} onClick={onUpdate}>{updating ? t(strings.reach.copy.updating) : t(strings.reach.copy.updateBrowser)}</Button><p className="text-xs text-app-muted-foreground">{t(strings.reach.copy.updateHelp)}</p></div>}
  </div>;
}

function channelLabel(channel: string, t: TFunction<"translation">): string {
  if (channel === "web_push") return t(strings.reach.copy.webPush);
  if (channel === "macos_notification") return t(strings.reach.copy.macNotification);
  if (channel === "imessage") return t(strings.reach.copy.imessage);
  if (channel === "email") return t(strings.reach.copy.email);
  return `${t(strings.reach.copy.otherChannel)} · ${channel}`;
}

function permissionLabel(permission: string, t: TFunction<"translation">): string {
  if (permission === "granted") return t(strings.reach.copy.permissionGranted);
  if (permission === "denied") return t(strings.reach.copy.permissionDeniedValue);
  if (permission === "default") return t(strings.reach.copy.permissionDefault);
  if (permission === "unsupported") return t(strings.reach.copy.unsupported);
  return t(strings.reach.copy.unknown);
}

function sensitivityLabel(label: string, t: TFunction<"translation">): string {
  if (label === "public" || label === "private" || label === "sensitive" || label === "critical") return t(strings.reach.copy[label]);
  return label;
}

function base64UrlToBytes(value: string): Uint8Array {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(value.length / 4) * 4, "=");
  const decoded = atob(normalized);
  return Uint8Array.from(decoded, (character) => character.charCodeAt(0));
}

function newStableAddressId(): string {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID();
  if (!globalThis.crypto?.getRandomValues) throw new Error("Secure random IDs are unavailable in this browser");
  const bytes = globalThis.crypto.getRandomValues(new Uint8Array(16));
  bytes[6] = ((bytes[6] ?? 0) & 0x0f) | 0x40;
  bytes[8] = ((bytes[8] ?? 0) & 0x3f) | 0x80;
  const hex = [...bytes].map((byte) => byte.toString(16).padStart(2, "0")).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}
