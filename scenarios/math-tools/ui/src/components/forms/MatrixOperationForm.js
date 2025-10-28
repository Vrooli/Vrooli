import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from 'react';
import { Grid3x3, Loader2 } from 'lucide-react';
import toast from 'react-hot-toast';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { safeJsonParse, stringify } from '@/lib/utils';
import { performCalculation } from '@/lib/api';
const MATRIX_OPERATIONS = [
    { value: 'matrix_multiply', label: 'Matrix Multiplication' },
    { value: 'matrix_inverse', label: 'Matrix Inverse' },
    { value: 'matrix_determinant', label: 'Determinant' },
    { value: 'matrix_transpose', label: 'Transpose' },
];
const DEFAULT_MATRIX_A = `[[1, 2],\n [3, 4]]`;
const DEFAULT_MATRIX_B = `[[5, 6],\n [7, 8]]`;
export function MatrixOperationForm({ settings }) {
    const [operation, setOperation] = useState('matrix_multiply');
    const [matrixAInput, setMatrixAInput] = useState(DEFAULT_MATRIX_A);
    const [matrixBInput, setMatrixBInput] = useState(DEFAULT_MATRIX_B);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [response, setResponse] = useState(null);
    async function handleSubmit(event) {
        event.preventDefault();
        const matrixA = safeJsonParse(matrixAInput);
        if (!matrixA) {
            toast.error('Matrix A must be valid JSON (e.g. [[1,2],[3,4]])');
            return;
        }
        const requiresMatrixB = operation === 'matrix_multiply';
        const matrixB = requiresMatrixB ? safeJsonParse(matrixBInput) : undefined;
        if (requiresMatrixB && !matrixB) {
            toast.error('Matrix B is required for multiplication and must be valid JSON');
            return;
        }
        try {
            setIsSubmitting(true);
            const result = await performCalculation({
                operation,
                matrix_a: matrixA,
                ...(matrixB ? { matrix_b: matrixB } : {}),
            }, settings);
            setResponse(result);
            toast.success('Matrix operation completed');
        }
        catch (error) {
            console.error(error);
            toast.error(error instanceof Error ? error.message : 'Matrix operation failed');
        }
        finally {
            setIsSubmitting(false);
        }
    }
    return (_jsxs(Card, { className: "h-full", children: [_jsxs(CardHeader, { className: "flex flex-col gap-3", children: [_jsxs(Badge, { variant: "secondary", className: "w-fit gap-1", children: [_jsx(Grid3x3, { className: "h-4 w-4" }), " Linear Algebra"] }), _jsx(CardTitle, { children: "Matrix Operations" }), _jsx(CardDescription, { children: "Work with matrix math for transformations and systems of equations." })] }), _jsxs(CardContent, { children: [_jsxs("form", { onSubmit: handleSubmit, className: "space-y-4", children: [_jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "matrix-operation", children: "Operation" }), _jsxs(Select, { value: operation, onValueChange: setOperation, children: [_jsx(SelectTrigger, { id: "matrix-operation", children: _jsx(SelectValue, { placeholder: "Choose matrix operation" }) }), _jsx(SelectContent, { children: MATRIX_OPERATIONS.map(option => (_jsx(SelectItem, { value: option.value, children: option.label }, option.value))) })] })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "matrix-a", children: "Matrix A" }), _jsx(Textarea, { id: "matrix-a", value: matrixAInput, onChange: event => setMatrixAInput(event.target.value), spellCheck: false })] }), operation === 'matrix_multiply' && (_jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "matrix-b", children: "Matrix B" }), _jsx(Textarea, { id: "matrix-b", value: matrixBInput, onChange: event => setMatrixBInput(event.target.value), spellCheck: false })] })), _jsx(Button, { type: "submit", className: "w-full", disabled: isSubmitting, children: isSubmitting ? (_jsxs("span", { className: "flex items-center gap-2", children: [_jsx(Loader2, { className: "h-4 w-4 animate-spin" }), "Solving\u2026"] })) : ('Run Matrix Operation') })] }), response && (_jsxs("div", { className: "mt-6 space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm", children: [_jsxs("div", { className: "flex items-center justify-between text-xs uppercase tracking-wide text-muted-foreground", children: [_jsx("span", { children: response.operation }), _jsxs("span", { children: [response.execution_time_ms, " ms"] })] }), _jsx("pre", { className: "max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs", children: stringify(response.result) })] }))] })] }));
}
