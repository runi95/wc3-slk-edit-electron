clean:
	rm -f bind.go
	rm -f bind_darwin_amd64.go
	rm -f bind_linux_amd64.go
	rm -f bind_windows_amd64.go
	astilectron-bundler cc -v

build:
	astilectron-bundler -v

start:
	"./output/linux-amd64/Warcraft_III_SLK_Edit"

debug:
	"./output/linux-amd64/Warcraft_III_SLK_Edit" -d