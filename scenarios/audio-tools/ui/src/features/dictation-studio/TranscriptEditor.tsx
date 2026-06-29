import { Textarea } from "../../components/ui/textarea";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";

interface Props {
  value: string;
  onChange: (value: string) => void;
}

// TranscriptEditor lets the operator correct the streamed batch transcript
// into the exact ground-truth reference text stored with the clip.
export function TranscriptEditor({ value, onChange }: Props) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.transcriptHint)}</p>
      <Textarea
        data-testid={selectors.dictationStudio.transcriptEditor}
        aria-label={t(strings.dictationStudio.transcriptTitle)}
        value={value}
        placeholder={t(strings.dictationStudio.transcriptPlaceholder)}
        onChange={(e) => onChange(e.currentTarget.value)}
        rows={4}
      />
    </div>
  );
}
