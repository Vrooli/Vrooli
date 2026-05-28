import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import {
  deleteSpeakerProfileClip,
  listSpeakerProfileClips,
} from "../../services/speakerAdmin";

// SpeakerProfileClips lists one profile's enrollment clips and lets the operator
// delete individual clips (deleting the last clip removes the profile).
export function SpeakerProfileClips({ profileId }: { profileId: string }) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const clips = useQuery({
    queryKey: ["speaker", "clips", profileId],
    queryFn: () => listSpeakerProfileClips(profileId),
  });
  const delMut = useMutation({
    mutationFn: (clipId: string) => deleteSpeakerProfileClip(profileId, clipId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["speaker", "clips", profileId] });
      void qc.invalidateQueries({ queryKey: ["speaker", "status"] });
    },
  });

  if (clips.isPending || clips.isError) {
    return null;
  }
  if (clips.data.length === 0) {
    return <p className="px-4 py-2 text-xs text-app-muted-foreground">{t(strings.speakerAdmin.clipsEmpty)}</p>;
  }

  return (
    <div className="px-4 py-2 text-xs">
      <p className="mb-1 font-medium">{t(strings.speakerAdmin.clipsTitle)}</p>
      <ul className="flex flex-col gap-1">
        {clips.data.map((c) => (
          <li
            key={c.clipId}
            data-testid={selectors.speakerAdmin.clipRow({ id: c.clipId })}
            className="flex items-center gap-2"
          >
            <span className="font-mono">{c.label || c.clipId}</span>
            <span className="text-app-muted-foreground">{`${c.voicedSeconds.toFixed(1)}s`}</span>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => delMut.mutate(c.clipId)}
            disabled={delMut.isPending}
            >
              {t(strings.speakerAdmin.deleteClipButton)}
            </Button>
          </li>
        ))}
      </ul>
    </div>
  );
}
