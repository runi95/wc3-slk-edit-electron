@ECHO OFF
DEL /Q bind.go >nul 2>&1
DEL /Q bind_darwin_amd64.go >nul 2>&1
DEL /Q bind_linux_amd64.go >nul 2>&1
DEL /Q bind_windows_amd64.go >nul 2>&1
astilectron-bundler cc -v