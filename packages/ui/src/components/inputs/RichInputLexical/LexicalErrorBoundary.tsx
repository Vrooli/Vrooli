/**
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 *
 */

import { Component } from "react";
import { ErrorBoundaryProps } from "views/types";

/**
 * Displays an error message if lexical throws an error.
 * 
 * NOTE: Cannot be a functional component. See https://legacy.reactjs.org/docs/error-boundaries.html#introducing-error-boundaries
 */
export class LexicalErrorBoundary extends Component<ErrorBoundaryProps> {
    render() {
        return (
            <div
                style={{
                    border: "1px solid #f00",
                    color: "#f00",
                    padding: "8px",
                }}>
                An error was thrown.
            </div>
        );
    }
}
