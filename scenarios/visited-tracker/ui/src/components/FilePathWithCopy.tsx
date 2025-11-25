import { Copy, Check } from 'lucide-react';
import { useClipboard } from '../hooks/useClipboard';
import { Button } from './ui/button';

interface FilePathWithCopyProps {
  path: string;
  className?: string;
}

export function FilePathWithCopy({ path, className = '' }: FilePathWithCopyProps) {
  const { isCopied, copy } = useClipboard();

  const handleCopy = (e: React.MouseEvent) => {
    e.stopPropagation();
    copy(path);
  };

  return (
    <div className={`flex items-center gap-2 group ${className}`}>
      <div className="font-mono text-sm text-slate-300 truncate flex-1" title={path}>
        {path}
      </div>
      <Button
        variant="ghost"
        size="sm"
        onClick={handleCopy}
        className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
        title="Copy path"
      >
        {isCopied ? (
          <Check className="h-3 w-3 text-green-400" />
        ) : (
          <Copy className="h-3 w-3 text-slate-400" />
        )}
      </Button>
    </div>
  );
}
