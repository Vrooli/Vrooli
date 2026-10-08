import { useState } from "react";
import { X } from "lucide-react";

import { Badge } from "../../../components/ui/badge";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";

interface Props {
  tags: string[];
  onChange: (tags: string[]) => void;
}

// TagInput renders the corpus tags as removable Badges with an Enter-to-add
// text input. Duplicate / blank tags are ignored so the corpus stays clean.
export function TagInput({ tags, onChange }: Props) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState("");

  const addTag = () => {
    const value = draft.trim();
    if (value === "" || tags.includes(value)) {
      setDraft("");
      return;
    }
    onChange([...tags, value]);
    setDraft("");
  };

  const removeTag = (tag: string) => {
    onChange(tags.filter((it) => it !== tag));
  };

  return (
    <div className="flex flex-col gap-2">
      {tags.length > 0 ? (
        <ul className="flex flex-wrap gap-2">
          {tags.map((tag) => (
            <li key={tag}>
              <Badge variant="info" className="gap-1">
                <span>{tag}</span>
                <button
                  type="button"
                  aria-label={t(strings.dictationStudio.removeTag)}
                  onClick={() => removeTag(tag)}
                  className="rounded-full p-0.5 hover:bg-app-surface-muted"
                >
                  <X className="h-3 w-3" aria-hidden="true" />
                </button>
              </Badge>
            </li>
          ))}
        </ul>
      ) : null}
      <div className="flex items-center gap-2">
        <Input
          data-testid={selectors.dictationStudio.tagInput}
          aria-label={t(strings.dictationStudio.addTag)}
          value={draft}
          placeholder={t(strings.dictationStudio.tagPlaceholder)}
          onChange={(e) => setDraft(e.currentTarget.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              addTag();
            }
          }}
        />
        <Button type="button" variant="outline" onClick={addTag} disabled={draft.trim() === ""}>
          {t(strings.dictationStudio.addTag)}
        </Button>
      </div>
    </div>
  );
}
