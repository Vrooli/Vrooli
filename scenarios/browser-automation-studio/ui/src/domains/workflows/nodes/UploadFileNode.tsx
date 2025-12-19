import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { UploadCloud, FileWarning } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useSyncedString, useSyncedNumber } from '@hooks/useSyncedField';
import { selectors } from '@constants/selectors';
import BaseNode from './BaseNode';
import { NodeTextField, NodeTextArea, NodeNumberField, FieldRow } from './fields';

// UploadFileParams interface for V2 native action params
interface UploadFileParams {
  selector?: string;
  filePaths?: string[];
  filePath?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const deriveFileText = (filePaths?: string[], filePath?: string): string => {
  if (filePaths && filePaths.length > 0) {
    return filePaths.join('\n');
  }
  if (filePath) {
    return filePath;
  }
  return '';
};

const parseFilePaths = (value: string): string[] => {
  return value
    .split(/\r?\n|,/)
    .map((entry) => entry.trim())
    .filter(Boolean);
};

const MIN_TIMEOUT = 500;

const UploadFileNode: FC<NodeProps> = ({ id, selected }) => {
  const { params, updateParams } = useActionParams<UploadFileParams>(id);

  // String field
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });

  // File paths as text (special handling for array)
  const fileText = useSyncedString(deriveFileText(params?.filePaths, params?.filePath), {
    trim: false,
    onCommit: (v) => {
      const parsed = parseFilePaths(v);
      if (parsed.length === 0) {
        updateParams({ filePaths: undefined, filePath: undefined });
      } else if (parsed.length === 1) {
        updateParams({ filePaths: parsed, filePath: parsed[0] });
      } else {
        updateParams({ filePaths: parsed, filePath: undefined });
      }
    },
  });

  // Number fields
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 30000, {
    min: MIN_TIMEOUT,
    fallback: 30000,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });
  const waitForMs = useSyncedNumber(params?.waitForMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ waitForMs: v || undefined }),
  });

  const activePaths = useMemo(() => parseFilePaths(fileText.value), [fileText.value]);

  return (
    <BaseNode selected={selected} icon={UploadCloud} iconClassName="text-pink-300" title="Upload File">
      <p className="text-[11px] text-gray-500 mb-3">Attach local files to file inputs</p>

      <div className="space-y-3 text-xs">
        <NodeTextField
          field={selector}
          label="Target selector"
          placeholder="#upload"
          description="Must point to an <input type='file'> element."
        />

        <div>
          <NodeTextArea
            field={fileText}
            label="File paths"
            placeholder="/home/me/Documents/example.pdf"
            rows={3}
          />
          <div className="mt-1 text-[10px] text-gray-500 flex items-center gap-1">
            <FileWarning size={12} className="text-amber-300" />
            Paths must be absolute on the execution machine. One per line for multiple files.
          </div>
          {activePaths.length > 0 && (
            <p
              className="text-[10px] text-emerald-300 mt-1"
              data-testid={selectors.workflowBuilder.nodes.upload.pathCount({ id })}
            >
              {activePaths.length === 1 ? '1 file selected' : `${activePaths.length} files selected`}
            </p>
          )}
        </div>

        <FieldRow>
          <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={MIN_TIMEOUT} />
          <NodeNumberField field={waitForMs} label="Wait after (ms)" min={0} />
        </FieldRow>
      </div>
    </BaseNode>
  );
};

export default memo(UploadFileNode);
