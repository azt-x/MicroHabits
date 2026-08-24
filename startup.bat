@echo off
setlocal

cd /d "%~dp0"

if not exist ".env" (
    echo ERROR: .env file was not found in the project folder.
    echo Create .env with DATABASE_PATH and JWT_SECRET before starting the server.
    pause
    exit /b 1
)

echo Starting MicroHabits API...
go run ./cmd/server

if errorlevel 1 (
    echo.
    echo The server stopped with an error.
    pause
)

endlocal
