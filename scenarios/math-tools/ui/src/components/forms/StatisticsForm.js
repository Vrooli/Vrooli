import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useMemo, useState } from 'react';
import { BarChart3, Loader2 } from 'lucide-react';
import toast from 'react-hot-toast';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { parseNumberList, stringify } from '@/lib/utils';
import { runStatistics } from '@/lib/api';
const AVAILABLE_ANALYSES = [
    { value: 'descriptive', label: 'Descriptive Statistics (mean, median, std dev)' },
    { value: 'correlation', label: 'Correlation Matrix' },
    { value: 'regression', label: 'Linear Regression' },
];
export function StatisticsForm({ settings }) {
    const [dataInput, setDataInput] = useState('1,2,3,4,5,6,7,8,9,10');
    const [selectedAnalyses, setSelectedAnalyses] = useState(['descriptive']);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [response, setResponse] = useState(null);
    const statisticsSummary = useMemo(() => {
        if (!response?.results) {
            return null;
        }
        const descriptive = response.results['descriptive'];
        if (!descriptive) {
            return null;
        }
        return [
            { label: 'Mean', value: descriptive['mean'] },
            { label: 'Median', value: descriptive['median'] },
            { label: 'Std Dev', value: descriptive['std_dev'] },
            { label: 'Variance', value: descriptive['variance'] },
            { label: 'Min', value: descriptive['min'] },
            { label: 'Max', value: descriptive['max'] },
        ];
    }, [response]);
    async function handleSubmit(event) {
        event.preventDefault();
        const data = parseNumberList(dataInput);
        if (data.length < 2) {
            toast.error('Statistics requires at least two values');
            return;
        }
        if (selectedAnalyses.length === 0) {
            toast.error('Select at least one analysis to run');
            return;
        }
        try {
            setIsSubmitting(true);
            const result = await runStatistics({
                data,
                analyses: selectedAnalyses,
            }, settings);
            setResponse(result);
            toast.success('Statistical analysis complete');
        }
        catch (error) {
            console.error(error);
            toast.error(error instanceof Error ? error.message : 'Failed to compute statistics');
        }
        finally {
            setIsSubmitting(false);
        }
    }
    function toggleAnalysis(value) {
        setSelectedAnalyses(prev => (prev.includes(value) ? prev.filter(item => item !== value) : [...prev, value]));
    }
    return (_jsxs(Card, { className: "h-full", children: [_jsxs(CardHeader, { className: "flex flex-col gap-3", children: [_jsxs(Badge, { variant: "secondary", className: "w-fit gap-1", children: [_jsx(BarChart3, { className: "h-4 w-4" }), " Statistical Insights"] }), _jsx(CardTitle, { children: "Run Statistical Analyses" }), _jsx(CardDescription, { children: "Upload datasets and compute descriptive statistics, correlation, or regression." })] }), _jsxs(CardContent, { children: [_jsxs("form", { onSubmit: handleSubmit, className: "space-y-4", children: [_jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "statistics-data", children: "Dataset" }), _jsx(Input, { id: "statistics-data", value: dataInput, onChange: event => setDataInput(event.target.value), placeholder: "Numbers separated by comma or space", autoComplete: "off" })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { children: "Analyses" }), _jsx(ScrollArea, { className: "h-[120px] w-full rounded-md border border-slate-800 bg-slate-900/40 p-3", children: _jsx("div", { className: "grid gap-3", children: AVAILABLE_ANALYSES.map(analysis => (_jsxs("label", { className: "flex items-center gap-3 text-sm text-slate-200", children: [_jsx(Checkbox, { checked: selectedAnalyses.includes(analysis.value), onCheckedChange: () => toggleAnalysis(analysis.value) }), _jsx("span", { children: analysis.label })] }, analysis.value))) }) })] }), _jsx(Button, { type: "submit", className: "w-full", disabled: isSubmitting, children: isSubmitting ? (_jsxs("span", { className: "flex items-center gap-2", children: [_jsx(Loader2, { className: "h-4 w-4 animate-spin" }), "Analysing\u2026"] })) : ('Run Analysis') })] }), response && (_jsxs("div", { className: "mt-6 space-y-4", children: [statisticsSummary && (_jsx("div", { className: "grid grid-cols-2 gap-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm md:grid-cols-3", children: statisticsSummary.map(stat => (_jsxs("div", { className: "flex flex-col", children: [_jsx("span", { className: "text-xs uppercase tracking-wide text-muted-foreground", children: stat.label }), _jsx("span", { className: "text-base font-semibold text-foreground", children: Number.isFinite(stat.value) ? stat.value?.toFixed(3) : '—' })] }, stat.label))) })), _jsxs("div", { className: "space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm", children: [_jsxs("div", { className: "flex items-center justify-between text-xs uppercase tracking-wide text-muted-foreground", children: [_jsx("span", { children: "Raw Response" }), _jsxs("span", { children: [response.data_points, " data points"] })] }), _jsx("pre", { className: "max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs", children: stringify(response.results) })] })] }))] })] }));
}
