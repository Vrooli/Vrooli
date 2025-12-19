import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { HardDriveDownload } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedSelect,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';
import {
  NodeTextField,
  NodeNumberField,
  NodeSelectField,
  FieldRow,
} from './fields';

// GetStorageParams interface for V2 native action params
interface GetStorageParams {
  storageType?: string;
  key?: string;
  resultFormat?: string;
  storeResult?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const STORAGE_OPTIONS = [
  { label: 'localStorage', value: 'localStorage' },
  { label: 'sessionStorage', value: 'sessionStorage' },
];

const RESULT_OPTIONS = [
  { label: 'Text', value: 'text' },
  { label: 'JSON (parse)', value: 'json' },
];

const MIN_TIMEOUT = 100;

const GetStorageNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<GetStorageParams>(id);

  // Select fields
  const storageType = useSyncedSelect(params?.storageType ?? 'localStorage', {
    onCommit: (v) => updateParams({ storageType: v }),
  });
  const resultFormat = useSyncedSelect(params?.resultFormat ?? 'text', {
    onCommit: (v) => updateParams({ resultFormat: v }),
  });

  // String fields
  const keyValue = useSyncedString(params?.key ?? '', {
    onCommit: (v) => updateParams({ key: v || undefined }),
  });
  const storeAs = useSyncedString(params?.storeResult ?? '', {
    onCommit: (v) => updateParams({ storeResult: v || undefined }),
  });

  // Number fields
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 15000, {
    min: MIN_TIMEOUT,
    fallback: 15000,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });
  const waitForMs = useSyncedNumber(params?.waitForMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ waitForMs: v || undefined }),
  });

  return (
    <BaseNode selected={selected} icon={HardDriveDownload} iconClassName="text-teal-300" title="Get Storage">
      <div className="space-y-3 text-xs">
        <FieldRow>
          <NodeSelectField field={storageType} label="Storage" options={STORAGE_OPTIONS} />
          <NodeTextField field={keyValue} label="Key" placeholder="profile" />
        </FieldRow>

        <FieldRow>
          <NodeSelectField field={resultFormat} label="Result format" options={RESULT_OPTIONS} />
          <NodeTextField field={storeAs} label="Store as" placeholder="profileData" />
        </FieldRow>

        <FieldRow>
          <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={MIN_TIMEOUT} />
          <NodeNumberField field={waitForMs} label="Post-read wait (ms)" min={0} />
        </FieldRow>
      </div>
    </BaseNode>
  );
};

export default memo(GetStorageNode);
