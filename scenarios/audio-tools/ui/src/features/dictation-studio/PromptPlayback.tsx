import { useRef } from "react";
import { Play } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Textarea } from "../../components/ui/textarea";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";

interface Props {
  prompt: string;
  onChange: (prompt: string) => void;
  readOnly?: boolean;
  /** Optional reference-audio URL (e.g. a TTS render) the operator can hear. */
  audioUrl?: string;
}

// PromptPlayback drives scripted-prompt capture: it shows the prompt the
// operator reads aloud and (when a reference URL is provided) plays it back
// through an HTMLAudioElement. The shown prompt becomes the clip's exact
// ground-truth reference text, and the page marks the source SCRIPTED.
export function PromptPlayback({ prompt, onChange, readOnly = false, audioUrl }: Props) {
  const { t } = useTranslation();
  const audioRef = useRef<HTMLAudioElement | null>(null);

  const play = () => {
    if (!audioUrl) return;
    if (audioRef.current === null) {
      audioRef.current = new Audio(audioUrl);
    }
    // Some environments reject media playback (autoplay policy, jsdom). The
    // reference render is a convenience, never load-bearing, so swallow it.
    void audioRef.current.play().catch(() => undefined);
  };

  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.scriptedPromptHint)}</p>
      <Textarea
        data-testid={selectors.dictationStudio.promptInput}
        aria-label={t(strings.dictationStudio.scriptedPromptTitle)}
        value={prompt}
        placeholder={t(strings.dictationStudio.promptPlaceholder)}
        readOnly={readOnly}
        onChange={(e) => onChange(e.currentTarget.value)}
        rows={3}
      />
      <div>
        <Button type="button" variant="outline" size="sm" onClick={play} disabled={!audioUrl}>
          <Play className="h-4 w-4" aria-hidden="true" />
          {t(strings.dictationStudio.playPrompt)}
        </Button>
      </div>
    </div>
  );
}
