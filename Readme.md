# Warcraft III SLK Editor

A simple editor that lets you edit Warcraft III SLK files much more easily.

## Installation

### Linux

Linux users will have to build the binaries themselves by following the steps below

1. Run `go get -u github.com/asticode/go-astilectron-bundler/...` to download the electron bundler
2. Run `go install github.com/asticode/go-astilectron-bundler/astilectron-bundler` to install the electron bundler
3. Run `go get -u github.com/runi95/wc3-slk-edit-electron` to download the SLK editor
4. Run `cd $GOPATH/src/github.com/runi95/wc3-slk-edit-electron` to enter the new directory
5. Run `make clean` to clean the workspace (this is required)
6. Run `make build` to build the application
7. Find your binary file inside `output/linux-amd64` or start it with `make start` (only recommended to use as a test)

### Windows

You can simply download the Warcraft_III_SLK_Edit.exe file from [latest releases](https://github.com/runi95/wc3-slk-edit-electron/releases/latest) or you can follow the steps below to build the executable yourself

1. Run `go get -u github.com/asticode/go-astilectron-bundler/...` to download the electron bundler
2. Run `go install github.com/asticode/go-astilectron-bundler/astilectron-bundler` to install the electron bundler
3. Run `go get -u github.com/runi95/wc3-slk-edit-electron` to download the SLK editor
4. Run `cd $GOPATH/src/github.com/runi95/wc3-slk-edit-electron` to enter the new directory
5. Run `clean.bat` to clean the workspace (this is required)
6. Run `build.bat` to start building
7. Find your executable inside `output\windows-amd64` or run with `run.bat` (only recommended to use as a test)

### Mac OS

Coming soon...

## Running the editor

If you've downloaded the executable or the binary file then everything should work out of the box. If you do not want to download the binaries then you can follow the steps above to compile the program yourself

## Advanced inputs

You can show or hide advanced inputs that are rarely used by clicking the :lock: and :unlock: icons at the top left corner.

## Preview

![Preview Image](/images/Preview-Image-1.png)
