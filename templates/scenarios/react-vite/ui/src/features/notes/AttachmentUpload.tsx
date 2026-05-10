import { useRef, useState, type ChangeEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Upload } from "lucide-react";

import { uploadAttachment } from "../../api/notes";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import {
  canStartAttachmentUpload,
  initialAttachmentUploadState,
  transitionAttachmentUpload,
  type AttachmentUploadState,
} from "./AttachmentUploadWorkflow";

const NOTES_QUERY_KEY = ["notes"] as const;

interface AttachmentUploadProps {
  noteId: string;
}

export function AttachmentUpload({ noteId }: AttachmentUploadProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [uploadState, setUploadState] = useState<AttachmentUploadState>(initialAttachmentUploadState);
  const nextAttemptId = useRef(0);

  const uploadMutation = useMutation({
    mutationFn: ({ file }: { file: File; attemptId: string }) => {
      return uploadAttachment(noteId, file);
    },
    onSuccess: (_attachment, { file, attemptId }) => {
      setUploadState((state) =>
        transitionAttachmentUpload(state, { type: "succeed", attemptId, fileName: file.name }),
      );
      void queryClient.invalidateQueries({ queryKey: NOTES_QUERY_KEY });
    },
    onError: (error, { attemptId }) => {
      setUploadState((state) =>
        transitionAttachmentUpload(state, { type: "fail", attemptId, message: errorMessage(error, t) }),
      );
    },
  });

  const handleFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const nextFile = event.target.files?.[0] ?? null;
    setUploadState((state) => nextFile
      ? transitionAttachmentUpload(state, { type: "select", file: nextFile })
      : transitionAttachmentUpload(state, { type: "reset" }));
  };

  const handleUpload = () => {
    if (!canStartAttachmentUpload(uploadState)) {
      return;
    }
    const file = uploadState.file;
    const attemptId = `upload-${nextAttemptId.current += 1}`;
    setUploadState((state) => transitionAttachmentUpload(state, { type: "start", attemptId }));
    uploadMutation.mutate({ file, attemptId });
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
        title={canStartAttachmentUpload(uploadState) ? undefined : t(strings.notes.noFileSelected)}
        disabled={!canStartAttachmentUpload(uploadState)}
        onClick={handleUpload}
      >
        <Upload aria-hidden="true" className="me-2 h-4 w-4" />
        {t(strings.notes.uploadAttachment)}
      </Button>
      {(uploadState.status === "failed" || uploadState.status === "succeeded") && (
        <p data-testid={selectors.notes.attachmentStatus} className="text-xs text-slate-300">
          {uploadState.status === "failed"
            ? uploadState.message
            : t(strings.notes.uploadSuccess, { name: uploadState.fileName })}
        </p>
      )}
    </div>
  );
}
