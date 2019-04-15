# Warcraft III SLK Editor

A simple editor that lets you edit Warcraft III SLK files much more easily.

## Installation

### Linux

1. Download the electron bundler by running `go get -u github.com/asticode/go-astilectron-bundler/...`
2. Install the electron bundler by running `go install github.com/asticode/go-astilectron-bundler/astilectron-bundler
`
3. Run `go get github.com/runi95/wc3-slk-edit-electron`
4. Enter the editor's directory `cd $GOPATH/src/github.com/runi95/wc3-slk-edit-electron`
5. Run `go mod vendor` to get all the dependencies required by this project
6. Run `make build` to start building
7. Find your binary inside `output/linux-amd64/Warcraft_III_SLK_Edit` or run with `make start`

### Windows

You can simply download the Warcraft_III_SLK_Edit.exe file from [latest releases](https://github.com/runi95/wc3-slk-edit-electron/releases/latest) or you can follow the steps below to build the executable yourself.
1. Download the electron bundler by running `go get -u github.com/asticode/go-astilectron-bundler/...`
2. Install the electron bundler by running `go install github.com/asticode/go-astilectron-bundler/astilectron-bundler
`
3. Run `go get github.com/runi95/wc3-slk-edit-electron`
4. Enter the editor's directory `cd %GOPATH%/src/github.com/runi95/wc3-slk-edit-electron`
5. Run `go mod vendor` to get all the dependencies required by this project
6. Run `build.bat` to start building
7. Find your executable inside `output\windows-amd64\Warcraft_III_SLK_Edit.exe` or run with `run.bat`

### Mac OS

Coming soon...

## Running the editor

If you've downloaded the executable or the binary file then everything should work out of the box. If you do not want to download the binaries then you can follow the steps above to compile the program yourself

## Advanced inputs

You can show or hide advanced inputs that are rarely used by clicking the :lock: and :unlock: icons at the top left corner.

## Preview

![Preview Image](/images/Preview-Image-1.png)
