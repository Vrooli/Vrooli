import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { HomeIcon, RefreshIcon } from "@local/icons";
import { Button, Stack, Typography } from "@mui/material";
import { Component } from "react";
import { stringifySearchParams } from "../../utils/route";
export class ErrorBoundary extends Component {
    constructor(props) {
        super(props);
        this.state = { hasError: false, error: null, mailToUrl: "mailto:official@vrooli.com" };
    }
    static getDerivedStateFromError(error) {
        const subject = `Error Report: ${error?.name ?? ""}`;
        const body = `Error message: ${error?.toString() ?? ""}`;
        const mailToUrl = `mailto:official@vrooli.com${stringifySearchParams({ body, subject })}`;
        return { hasError: true, error, mailToUrl };
    }
    handleRefresh = () => {
        window.location.reload();
    };
    render() {
        const { hasError, error, mailToUrl } = this.state;
        if (hasError) {
            return (_jsx("div", { style: {
                    display: "flex",
                    justifyContent: "center",
                    alignItems: "center",
                    height: "100%",
                    position: "absolute",
                    top: 0,
                    left: 0,
                    right: 0,
                    bottom: 0,
                    paddingLeft: "16px",
                    paddingRight: "16px",
                    backgroundColor: "white",
                    color: "black",
                }, children: _jsxs(Stack, { direction: "column", spacing: 2, style: { textAlign: "center" }, children: [_jsx(Typography, { variant: "h4", children: "Something went wrong \uD83D\uDE14" }), _jsx(Typography, { variant: "body1", style: { color: "red" }, children: error?.toString() ?? "" }), _jsxs(Typography, { variant: "body1", children: ["Please try refreshing the page. If the problem persists, you can", " ", _jsx("a", { href: mailToUrl, target: "_blank", rel: "noopener noreferrer", children: "contact us" }), " ", "and we will try to help you as soon as possible."] }), _jsx(Button, { variant: "contained", startIcon: _jsx(RefreshIcon, {}), onClick: this.handleRefresh, sx: { marginTop: "16px" }, children: "Refresh" }), _jsx(Button, { variant: "contained", startIcon: _jsx(HomeIcon, {}), onClick: () => window.location.assign("/"), sx: { marginTop: "16px" }, children: "Go to Home" })] }) }));
        }
        return this.props.children;
    }
}
//# sourceMappingURL=ErrorBoundary.js.map