import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { HardDriveUpload } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedSelect,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';
import {
  NodeTextField,
  NodeTextArea,
  NodeNumberField,
  NodeSelectField,
  FieldRow,
} from './fields';

// SetStorageParams interface for V2 native action params
interface SetStorageParams {
  storageType?: string;
  key?: string;
  value?: string;
  valueType?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const STORAGE_OPTIONS = [
  { label: 'localStorage', value: 'localStorage' },
  { label: 'sessionStorage', value: 'sessionStorage' },
];

const VALUE_TYPE_OPTIONS = [
  { label: 'Text', value: 'text' },
  { label: 'JSON', value: 'json' },
];

const MIN_TIMEOUT = 100;

const SetStorageNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<SetStorageParams>(id);

  // Select fields
  const storageType = useSyncedSelect(params?.storageType ?? 'localStorage', {
    onCommit: (v) => updateParams({ storageType: v }),
  });
  const valueType = useSyncedSelect(params?.valueType ?? 'text', {
    onCommit: (v) => updateParams({ valueType: v }),
  });

  // String fields
  const keyValue = useSyncedString(params?.key ?? '', {
    onCommit: (v) => updateParams({ key: v || undefined }),
  });
  const value = useSyncedString(params?.value ?? '', {
    trim: false, // Don't trim storage value
    onCommit: (v) => updateParams({ value: v }),
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
    <BaseNode selected={selected} icon={HardDriveUpload} iconClassName="text-violet-300" title="Set Storage">
      <div className="space-y-3 text-xs">
        <FieldRow>
          <NodeSelectField field={storageType} label="Storage" options={STORAGE_OPTIONS} />
          <NodeTextField field={keyValue} label="Key" placeholder="profile" />
        </FieldRow>

        <NodeSelectField field={valueType} label="Value type" options={VALUE_TYPE_OPTIONS} />

        <NodeTextArea
          field={value}
          label="Value"
          placeholder={valueType.value === 'json' ? '{"theme":"dark"}' : 'value'}
          rows={3}
          description={valueType.value === 'json' ? 'Must be valid JSON and will be stringified before storage.' : undefined}
        />

        <FieldRow>
          <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={MIN_TIMEOUT} />
          <NodeNumberField field={waitForMs} label="Post-set wait (ms)" min={0} />
        </FieldRow>
      </div>
    </BaseNode>
  );
};

export default memo(SetStorageNode);
