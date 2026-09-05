import { useRef, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Panel } from "../../../components/ui/panel";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";
import {
  enrollSpeakerProfile,
  type SpeakerEnrollmentResult,
} from "../../../services/speakerAdmin";
import {
  recordEnrollmentClip,
  type EnrollRecordHandle,
} from "../speakerEnrollmentRecorder";

export interface SpeakerEnrollmentPanelProps {
  // Injectable recorder seam so the panel is testable without a real
  // MediaRecorder (jsdom has none).
  recordClip?: typeof recordEnrollmentClip;
}

// SpeakerEnrollmentPanel records enrollment clips and appends each to one
// profile, building a multi-condition identity. The first clip creates the
// profile (server-generated id when blank); later clips reuse the returned id.
export function SpeakerEnrollmentPanel({ recordClip = recordEnrollmentClip }: SpeakerEnrollmentPanelProps) {
  const { t } = useTranslation();
  const qc = useQueryClient();

  const [displayName, setDisplayName] = useState("");
  const [profileId, setProfileId] = useState("");
  const [label, setLabel] = useState("");
  const [activate, setActivate] = useState(false);
  const [recording, setRecording] = useState(false);
  const [last, setLast] = useState<SpeakerEnrollmentResult | null>(null);
  const handleRef = useRef<EnrollRecordHandle | null>(null);

  const enrollMut = useMutation({
    mutationFn: enrollSpeakerProfile,
    onSuccess: (res) => {
      setLast(res);
      // First clip may return a server-generated id; reuse it for the next.
      setProfileId(res.profileId);
      void qc.invalidateQueries({ queryKey: ["speaker", "status"] });
    },
  });

  const startRecording = async () => {
    setRecording(true);
    const handle = await recordClip();
    handleRef.current = handle;
    const clip = await handle.done;
    setRecording(false);
    handleRef.current = null;
    enrollMut.mutate({
      audio: clip.audio,
      format: clip.format,
      profileId: profileId || undefined,
      displayName: displayName || undefined,
      label: label || undefined,
      addToActive: activate || undefined,
      enable: activate || undefined,
    });
  };

  const stopRecording = () => {
    handleRef.current?.stop();
  };

  return (
    <Panel title={t(strings.speakerAdmin.enrollTitle)}>
      <div className="flex flex-col gap-3 px-4 py-3 text-sm">
        <p className="text-xs text-app-muted-foreground">{t(strings.speakerAdmin.enrollHint)}</p>
        <label className="flex flex-col gap-1">
          {t(strings.speakerAdmin.enrollNameLabel)}
          <Input value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
        </label>
        <label className="flex flex-col gap-1">
          {t(strings.speakerAdmin.enrollProfileIdLabel)}
          <Input value={profileId} onChange={(e) => setProfileId(e.target.value)} />
        </label>
        <label className="flex flex-col gap-1">
          {t(strings.speakerAdmin.enrollClipLabelLabel)}
          <Input
            data-testid="speaker-enroll-clip-label"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
          />
          <span className="text-xs text-app-muted-foreground">{t(strings.speakerAdmin.enrollClipLabelHelp)}</span>
        </label>
        <label className="flex items-center gap-2">
          <input type="checkbox" checked={activate} onChange={(e) => setActivate(e.target.checked)} />
          {t(strings.speakerAdmin.enrollActivate)}
        </label>
        <div className="flex items-center gap-2">
          {recording ? (
            <Button type="button" variant="ghost" onClick={stopRecording}>
              {t(strings.speakerAdmin.enrollStop)}
            </Button>
          ) : (
            <Button
              type="button"
              data-testid="speaker-enroll-record"
              onClick={() => void startRecording()}
              disabled={enrollMut.isPending}
            >
              {t(strings.speakerAdmin.enrollRecord)}
            </Button>
          )}
          {recording && <span className="text-xs text-app-muted-foreground">{t(strings.speakerAdmin.enrollRecording)}</span>}
        </div>
        {last && (
          <div data-testid="speaker-enroll-last-clip" className="text-xs text-app-muted-foreground">
            {t(strings.speakerAdmin.enrollLastClip)}:{" "}
            {`${last.label || "—"} · ${last.voicedSeconds.toFixed(1)}s · ${last.clipCount}×`}
          </div>
        )}
      </div>
    </Panel>
  );
}
