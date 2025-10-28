import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useMemo, useState } from 'react';
import { LineChart, Loader2 } from 'lucide-react';
import toast from 'react-hot-toast';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';
import { parseNumberList, stringify } from '@/lib/utils';
import { forecast } from '@/lib/api';
const METHODS = [
    { value: 'linear_trend', label: 'Linear Trend' },
    { value: 'exponential_smoothing', label: 'Exponential Smoothing' },
    { value: 'moving_average', label: 'Moving Average' },
];
export function ForecastForm({ settings }) {
    const [seriesInput, setSeriesInput] = useState('100,102,98,105,110,108,112,115,118,120');
    const [horizon, setHorizon] = useState('5');
    const [method, setMethod] = useState('linear_trend');
    const [includeIntervals, setIncludeIntervals] = useState(true);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [response, setResponse] = useState(null);
    const summaryMetrics = useMemo(() => {
        if (!response?.model_metrics) {
            return [];
        }
        return Object.entries(response.model_metrics).map(([key, value]) => ({
            label: key.replace(/_/g, ' '),
            value,
        }));
    }, [response]);
    async function handleSubmit(event) {
        event.preventDefault();
        const data = parseNumberList(seriesInput);
        if (data.length < 3) {
            toast.error('Provide at least three data points for forecasting');
            return;
        }
        try {
            setIsSubmitting(true);
            const result = await forecast({
                time_series: data,
                forecast_horizon: Number.parseInt(horizon, 10) || 5,
                method,
                options: {
                    confidence_intervals: includeIntervals,
                    seasonality: false,
                    validation_split: 0.2,
                },
            }, settings);
            setResponse(result);
            toast.success('Forecast ready');
        }
        catch (error) {
            console.error(error);
            toast.error(error instanceof Error ? error.message : 'Forecasting failed');
        }
        finally {
            setIsSubmitting(false);
        }
    }
    return (_jsxs(Card, { className: "h-full", children: [_jsxs(CardHeader, { className: "flex flex-col gap-3", children: [_jsxs(Badge, { variant: "secondary", className: "w-fit gap-1", children: [_jsx(LineChart, { className: "h-4 w-4" }), " Forecasting"] }), _jsx(CardTitle, { children: "Time-Series Forecasting" }), _jsx(CardDescription, { children: "Predict future values with configurable statistical forecasting methods." })] }), _jsxs(CardContent, { children: [_jsxs("form", { onSubmit: handleSubmit, className: "space-y-4", children: [_jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "forecast-series", children: "Historical Series" }), _jsx(Input, { id: "forecast-series", value: seriesInput, onChange: event => setSeriesInput(event.target.value), placeholder: "Comma-separated values" })] }), _jsxs("div", { className: "grid gap-3 md:grid-cols-3", children: [_jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "forecast-horizon", children: "Horizon" }), _jsx(Input, { id: "forecast-horizon", value: horizon, onChange: event => setHorizon(event.target.value), placeholder: "5" })] }), _jsxs("div", { className: "grid gap-2 md:col-span-2", children: [_jsx(Label, { htmlFor: "forecast-method", children: "Method" }), _jsxs(Select, { value: method, onValueChange: setMethod, children: [_jsx(SelectTrigger, { id: "forecast-method", children: _jsx(SelectValue, { placeholder: "Select method" }) }), _jsx(SelectContent, { children: METHODS.map(item => (_jsx(SelectItem, { value: item.value, children: item.label }, item.value))) })] })] })] }), _jsxs("label", { className: "flex items-center gap-3 text-sm text-slate-200", children: [_jsx(Checkbox, { checked: includeIntervals, onCheckedChange: checked => setIncludeIntervals(Boolean(checked)) }), "Include 95% confidence interval bands"] }), _jsx(Button, { type: "submit", className: "w-full", disabled: isSubmitting, children: isSubmitting ? (_jsxs("span", { className: "flex items-center gap-2", children: [_jsx(Loader2, { className: "h-4 w-4 animate-spin" }), "Forecasting\u2026"] })) : ('Generate Forecast') })] }), response && (_jsxs("div", { className: "mt-6 space-y-4", children: [_jsx("div", { className: "grid gap-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm md:grid-cols-2", children: summaryMetrics.map(metric => (_jsxs("div", { className: "flex flex-col", children: [_jsx("span", { className: "text-xs uppercase tracking-wide text-muted-foreground", children: metric.label }), _jsx("span", { className: "text-base font-semibold text-foreground", children: metric.value.toFixed(3) })] }, metric.label))) }), _jsxs("div", { className: "space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm", children: [_jsx("div", { className: "text-xs uppercase tracking-wide text-muted-foreground", children: "Forecasted Values" }), _jsx("pre", { className: "max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs", children: stringify(response.forecast) })] }), includeIntervals && response.confidence_intervals && (_jsxs("div", { className: "grid gap-4 md:grid-cols-2", children: [_jsxs("div", { className: "space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm", children: [_jsx("div", { className: "text-xs uppercase tracking-wide text-muted-foreground", children: "Lower Bounds" }), _jsx("pre", { className: "max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs", children: stringify(response.confidence_intervals.lower) })] }), _jsxs("div", { className: "space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm", children: [_jsx("div", { className: "text-xs uppercase tracking-wide text-muted-foreground", children: "Upper Bounds" }), _jsx("pre", { className: "max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs", children: stringify(response.confidence_intervals.upper) })] })] }))] }))] })] }));
}
