import { selectors } from '@constants/selectors';

type MonacoEditorProps = {
  value: string;
  onChange: (value?: string) => void;
};

export default function MonacoEditorMock({ value, onChange }: MonacoEditorProps) {
  return (
    <textarea
      data-testid={selectors.workflowBuilder.monacoEditor}
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  );
}
