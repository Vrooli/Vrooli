import { Eraser, RefreshCw, UploadCloud, Wand2 } from 'lucide-react';
import * as React from 'react';

import type { SampleDataKey } from '../../lib/sample-data';
import { cn } from '../../lib/utils';
import { Button } from '../ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card';
import { Textarea } from '../ui/textarea';

const sampleLabels: Record<SampleDataKey, string> = {
  sales: 'Quarterly sales',
  revenue: 'Revenue by quarter',
  performance: 'Team performance',
  growth: 'Year-over-year growth',
  correlation: 'Risk vs reward',
};

interface DataPanelProps {
  value: string;
  onChange: (value: string) => void;
  onSampleSelect: (sample: SampleDataKey) => void;
  onValidate: () => void;
  onClear: () => void;
  onImport: (file: File) => Promise<void>;
  samples: SampleDataKey[];
  dataPoints: number;
  lastUpdated?: Date | null;
  isValid?: boolean | null;
  errors?: string | null;
}

export const DataPanel: React.FC<DataPanelProps> = ({
  value,
  onChange,
  onSampleSelect,
  onValidate,
  onClear,
  onImport,
  samples,
  dataPoints,
  lastUpdated,
  isValid,
  errors,
}) => {
  const inputRef = React.useRef<HTMLInputElement | null>(null);

  const handleUploadClick = () => {
    inputRef.current?.click();
  };

  const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      await onImport(file);
      event.target.value = '';
    }
  };

  return (
    <Card className="h-full bg-white/90">
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Data</CardTitle>
        <CardDescription>Paste JSON data, import a file, or start from a curated sample set.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-wrap items-center gap-2">
          {samples.map((sample) => (
            <Button
              key={sample}
              variant="outline"
              size="sm"
              onClick={() => onSampleSelect(sample)}
              data-testid="sample-data"
            >
              <Wand2 className="mr-1.5 h-4 w-4" />
              {sampleLabels[sample]}
            </Button>
          ))}
        </div>

        <Textarea
          spellCheck={false}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          data-testid="data-input"
          className={cn('font-mono text-xs leading-relaxed', isValid === false && 'border-red-400 focus-visible:ring-red-400')}
          placeholder='[
  { "x": "Jan", "y": 4200 },
  { "x": "Feb", "y": 5100 }
]'
        />

        <div className="flex flex-wrap items-center gap-2">
          <Button variant="secondary" size="sm" onClick={onValidate} data-testid="data-validate">
            <RefreshCw className="mr-1.5 h-4 w-4" /> Validate data
          </Button>
          <Button variant="ghost" size="sm" onClick={onClear}>
            <Eraser className="mr-1.5 h-4 w-4" /> Clear
          </Button>
          <Button variant="outline" size="sm" onClick={handleUploadClick}>
            <UploadCloud className="mr-1.5 h-4 w-4" /> Import JSON
          </Button>
          <input
            ref={inputRef}
            type="file"
            accept="application/json,.json"
            className="hidden"
            onChange={handleFileChange}
          />
        </div>

        <div className="flex flex-wrap items-center gap-6 rounded-lg border border-dashed border-border/80 bg-muted/40 px-4 py-3 text-sm">
          <span className="font-medium text-foreground">{dataPoints} data points</span>
          <span className="text-muted-foreground">
            {lastUpdated ? `Updated ${lastUpdated.toLocaleTimeString()}` : 'Not validated yet'}
          </span>
          {isValid === true && <span className="text-emerald-600">Data ready for charting</span>}
          {isValid === false && errors && <span className="text-red-500" data-testid="data-error">{errors}</span>}
        </div>
      </CardContent>
    </Card>
  );
};
