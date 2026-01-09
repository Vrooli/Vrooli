@echo off
REM Vrooli Autoheal Windows Batch Wrapper
REM This script provides a Windows-compatible entry point for the autoheal loop
REM
REM Usage: vrooli-autoheal.bat loop
REM        vrooli-autoheal.bat status
REM
REM The Go binary should be built and placed alongside this script:
REM   cd cli\loop && go build -o ..\vrooli-autoheal-loop.exe .

setlocal enabledelayedexpansion

REM Determine script directory
set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

REM Set VROOLI_ROOT if not already set
if not defined VROOLI_ROOT (
    for %%I in ("%SCRIPT_DIR%\..\..\..") do set "VROOLI_ROOT=%%~fI"
)

REM Set environment variables
set "VROOLI_LIFECYCLE_MANAGED=true"

REM Parse command
set "CMD=%~1"
if "%CMD%"=="" (
    echo Usage: vrooli-autoheal.bat [loop^|status^|help]
    exit /b 1
)

if /i "%CMD%"=="loop" (
    REM Check if Go binary exists
    if exist "%SCRIPT_DIR%\vrooli-autoheal-loop.exe" (
        "%SCRIPT_DIR%\vrooli-autoheal-loop.exe" %*
    ) else (
        echo Error: vrooli-autoheal-loop.exe not found
        echo Please build it with: go build -o vrooli-autoheal-loop.exe ./cli/loop
        exit /b 1
    )
) else if /i "%CMD%"=="status" (
    REM Check API health - auto-detect port from registry
    set "PORT_FILE=%USERPROFILE%\.vrooli\processes\scenarios\vrooli-autoheal\port"
    if exist "!PORT_FILE!" (
        set /p API_PORT=<"!PORT_FILE!"
        curl -s http://localhost:!API_PORT!/health
    ) else (
        REM Fall back to vrooli CLI
        vrooli scenario port vrooli-autoheal API_PORT >nul 2>&1
        if errorlevel 1 (
            echo Error: Could not detect API port
            echo Is the scenario running? Try: vrooli scenario start vrooli-autoheal
            exit /b 1
        )
        for /f %%p in ('vrooli scenario port vrooli-autoheal API_PORT 2^>nul') do set "API_PORT=%%p"
        if defined API_PORT (
            curl -s http://localhost:!API_PORT!/health
        ) else (
            echo Error: Could not detect API port
            exit /b 1
        )
    )
) else if /i "%CMD%"=="help" (
    echo Vrooli Autoheal CLI for Windows
    echo.
    echo Commands:
    echo   loop    Start the autoheal loop (for boot recovery^)
    echo   status  Check API health status
    echo   help    Show this help message
    echo.
    echo Environment:
    echo   VROOLI_ROOT=%VROOLI_ROOT%
    echo.
    echo Note: API_PORT is auto-detected from vrooli process registry
) else (
    echo Unknown command: %CMD%
    echo Run 'vrooli-autoheal.bat help' for usage
    exit /b 1
)

endlocal
