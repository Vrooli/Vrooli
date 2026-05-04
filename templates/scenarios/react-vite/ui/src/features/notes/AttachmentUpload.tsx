import { useState, type ChangeEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Upload } from "lucide-react";

import { uploadAttachment } from "../../api/notes";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

const NOTES_QUERY_KEY = ["notes"] as const;

interface AttachmentUploadProps {
  noteId: string;
}

export function AttachmentUpload({ noteId }: AttachmentUploadProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [file, setFile] = useState<File | null>(null);
  const [lastUploadedName, setLastUploadedName] = useState<string | null>(null);

  const uploadMutation = useMutation({
    mutationFn: () => {
      if (!file) {
        throw new Error(t(strings.notes.noFileSelected));
      }
      return uploadAttachment(noteId, file);
    },
    onSuccess: () => {
      setLastUploadedName(file?.name ?? null);
      setFile(null);
      void queryClient.invalidateQueries({ queryKey: NOTES_QUERY_KEY });
    },
  });

  const handleFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    setFile(event.target.files?.[0] ?? null);
    setLastUploadedName(null);
  };

  return (
    <div data-testid={selectors.notes.attachmentUpload} className="mt-3 flex flex-col gap-2">
      <input
        data-testid={selectors.notes.attachmentFile}
        aria-label={t(strings.notes.attachmentFileLabel)}
        className="block w-full text-xs text-slate-300 file:me-3 file:rounded-md file:border-0 file:bg-white/10 file:px-3 file:py-1.5 file:text-xs file:font-medium file:text-slate-100"
        type="file"
        onChange={handleFileChange}
      />
      <Button
        data-testid={selectors.notes.attachmentButton}
        type="button"
        className="w-fit"
        disabled={!file || uploadMutation.isPending}
        onClick={() => uploadMutation.mutate()}
      >
        <Upload aria-hidden="true" className="me-2 h-4 w-4" />
        {t(strings.notes.uploadAttachment)}
      </Button>
      {(uploadMutation.error || lastUploadedName) && (
        <p data-testid={selectors.notes.attachmentStatus} className="text-xs text-slate-300">
          {uploadMutation.error
            ? errorMessage(uploadMutation.error, t)
            : t(strings.notes.uploadSuccess, { name: lastUploadedName })}
        </p>
      )}
    </div>
  );
}
